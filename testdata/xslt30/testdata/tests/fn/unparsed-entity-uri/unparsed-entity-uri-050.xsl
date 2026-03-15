<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Inline Style are used for the simplicity of the test; They are not used in the production environment -->
<xsl:variable name="htmlbody">
	<xsl:text>display:block;</xsl:text>
</xsl:variable>

<xsl:variable name="Section1">
	<xsl:text>display:block;</xsl:text>
</xsl:variable>

<xsl:variable name="Section1Heading">
	<xsl:text>display:block;font-size: 150%;margin-bottom:12px;</xsl:text>
</xsl:variable>

<xsl:variable name="Para">
	<xsl:text>display:block;margin-bottom:6px;</xsl:text>
</xsl:variable>

<!-- End of Styles -->

<xsl:variable name="rootLangLoc" select="(/*[1]/@xml:lang)[1]"/>

<!-- ROOT ELEMENT -->
<xsl:template match="/">
	<xsl:variable name="title" select="'Entity Resolution Compliance Test'"/>
	<xsl:element name="html">
		<xsl:element name="title"><xsl:value-of select="$title"/></xsl:element>

		<xsl:element name="body">
			<xsl:attribute name="style"><xsl:value-of select="$htmlbody"/></xsl:attribute>
			<xsl:apply-templates/>
		</xsl:element>
	</xsl:element>
</xsl:template> <!-- match="Root" -->


<xsl:template match="Section1">
	<xsl:element name="div">
		<xsl:attribute name="style"><xsl:value-of select="$Section1"/></xsl:attribute>
		<xsl:apply-templates/>
	</xsl:element>
</xsl:template>


<xsl:template match="Section1/Heading">
	<xsl:element name="div">
		<xsl:attribute name="style"><xsl:value-of select="$Section1Heading"/></xsl:attribute>
		<xsl:apply-templates/>
	</xsl:element>
</xsl:template>


<xsl:template match="Para">
	<xsl:element name="div">
		<xsl:attribute name="style"><xsl:value-of select="$Para"/></xsl:attribute>
		<xsl:apply-templates/>
	</xsl:element>
</xsl:template>

<xsl:template match="MediaObject">
	<!-- Electronic Publications uses their own paths -->
	<xsl:variable name="imagesPath" select="'images\epub\'"/>
	<xsl:variable name="src_filename"> <!-- Get the image filename from the entity  -->
		<xsl:call-template name="strip_dir">
			<xsl:with-param name="string" select="unparsed-entity-uri(./@src)"/>
		</xsl:call-template>
	</xsl:variable>

	<xsl:element name="img">
		<xsl:attribute name="src"><xsl:value-of select="concat($imagesPath, $src_filename)"/></xsl:attribute>
	</xsl:element>
</xsl:template>

<xsl:template name="strip_dir">
	<xsl:param name="string"/>
	<xsl:choose>
		<xsl:when test="contains($string, '/')">
			<xsl:call-template name="strip_dir">
				<xsl:with-param name="string" select="substring-after($string, '/')"/>
			</xsl:call-template>
		</xsl:when>
		<xsl:when test="contains($string, '\')">
			<xsl:call-template name="strip_dir">
				<xsl:with-param name="string" select="substring-after($string, '\')"/>
			</xsl:call-template>
		</xsl:when>
		<xsl:otherwise>
			<xsl:value-of select="$string"/>
		</xsl:otherwise>
	</xsl:choose>
</xsl:template>



</xsl:stylesheet>

