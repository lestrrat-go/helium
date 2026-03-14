<!-- bug report from Jonathan Williford 
     last() fails when applied to an axis on a virtual copy of a node -->
     
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:fn="http://www.w3.org/2005/xpath-functions"
	version="2.0" xmlns:m2m="http://www.gh-mathspeak.com/m2m"
	xmlns:ir="http://www.gh-mathspeak.com/mathspeak-internal-representation"
	xmlns:m="http://www.w3.org/1998/Math/MathML" exclude-result-prefixes="m2m xs fn m"
	input-type-annotations="strip">


	<xsl:strip-space elements="*"/>

	<xsl:template match="m:math" mode="m2m_phase1">
		<ir:math>
			<xsl:apply-templates mode="m2m_phase1"/>
		</ir:math>
	</xsl:template>


	<xsl:template match="m:msup | m:msub | m:msubsup | m:mmultiscripts" mode="m2m_phase1">
		<xsl:variable name="supers" select="m2m:getSuperScript( . )"/>
		<ir:multiscripts>
			<ir:base>
				<xsl:apply-templates select="m2m:getBaseScript( . )" mode="m2m_phase1"/>
			</ir:base>
			<xsl:for-each select="$supers">
				<ir:superscript>
					<xsl:apply-templates mode="m2m_phase1" select=" . "> </xsl:apply-templates>
				</ir:superscript>
			</xsl:for-each>
			<xsl:if test=" $supers">
				<ir:base-line/>
			</xsl:if>
		</ir:multiscripts>
	</xsl:template>


	<xsl:function name="m2m:getBaseScript">
		<xsl:param name="currnode"/>
		<xsl:choose>
			<xsl:when test="$currnode/self::m:msup">
				<xsl:copy-of select="$currnode/*[1]"/>
			</xsl:when>
			<xsl:when test="$currnode/self::m:msub">
				<xsl:copy-of select="$currnode/*[1]"/>
			</xsl:when>
			<xsl:when test="$currnode/self::m:msubsup">
				<xsl:copy-of select="$currnode/*[1]"/>
			</xsl:when>
			<xsl:when test="$currnode/self::m:mmultiscripts">
				<xsl:copy-of select="$currnode/*[1]"/>
			</xsl:when>
			<xsl:otherwise>NOT YET IMPLEMENTED (getBaseScript) <xsl:value-of
					select="fn:name($currnode)"/>
			</xsl:otherwise>
		</xsl:choose>
	</xsl:function>

	<xsl:function name="m2m:getSuperScript">
		<xsl:param name="currnode"/>
		<xsl:choose>
			<xsl:when test="$currnode/self::m:msub"/>
			<xsl:when test="$currnode/self::m:msup">
				<xsl:copy-of select="$currnode/*[2]"/>
			</xsl:when>
			<xsl:when test="$currnode/self::m:mmultiscripts">
				<xsl:copy-of
					select="$currnode/m:mprescripts/preceding-sibling::*
						  [ (fn:position() mod 2 eq 1) and (fn:position() ne fn:last()) and (not(self::m:none))]"
				/>
			</xsl:when>
			<xsl:when test="$currnode/self::m:msubsup">
				<xsl:copy-of select="$currnode/*[3]"/>
			</xsl:when>
			<xsl:otherwise>NOT YET IMPLEMENTED (getSuperScript) <xsl:value-of
					select="fn:name($currnode)"/>
			</xsl:otherwise>
		</xsl:choose>
	</xsl:function>

	<xsl:variable name="intermediate">
		<xsl:apply-templates select="/" mode="m2m_phase1"/>
	</xsl:variable>

	<xsl:template match="/">
		<xsl:result-document>
			<xsl:copy-of select="$intermediate"/>
		</xsl:result-document>
	</xsl:template>


</xsl:transform>
