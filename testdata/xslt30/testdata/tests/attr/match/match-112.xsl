<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- Test of pattern matching of various parentless copied nodes -->

<xsl:template match="doc">

  	<xsl:variable name="elem" as="element()">
	  <xsl:copy-of select="element()"/>
	</xsl:variable>
	
	<xsl:variable name="pi1" as="processing-instruction()+">
	  <xsl:copy-of select="processing-instruction()"/>
	</xsl:variable>
	
	<xsl:variable name="comm" as="comment()">
	  <xsl:copy-of select="comment()"/>
	</xsl:variable>
	
	<xsl:variable name="txt" as="text()">
	  <xsl:copy-of select="text()"/>
	</xsl:variable>
	
	<xsl:variable name="att" as="attribute()">
		<xsl:copy-of select="attribute()"/>
	</xsl:variable>
  	
  	<out>
	  	<xsl:apply-templates select="$elem"/>
		<xsl:apply-templates select="$pi1"/>
		<xsl:apply-templates select="$comm"/>
	  	<xsl:apply-templates select="$txt"/>
	  	<xsl:apply-templates select="$att"/>
  	</out>
</xsl:template>

<xsl:template match="child::element()">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

<xsl:template match="child::processing-instruction()">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

<xsl:template match="child::text()">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

<xsl:template match="child::comment()">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

<xsl:template match="child::attribute()">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>


</xsl:stylesheet>
