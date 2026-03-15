<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#keys?>
  <!-- Section: 5.2 -->
    <!-- Purpose: Test complex key()//x/x, with complex predicate in match pattern. -->


<xsl:key name="Info" match="Level1 | Level2 | Level3" use="@ID"/>

<xsl:template match="/">
 <out>
	<xsl:apply-templates select="//Level3"/>
 </out>
</xsl:template>

<xsl:template match="key('Info','id1')//Level3/Level3[Name/@MI='Q'][Age='8']">
	<xsl:text>&#10;</xsl:text>
	<xsl:element name="Complex-Match">
		<xsl:value-of select="Name/@First"/>
	</xsl:element>
</xsl:template>


<xsl:template match="*">
	<xsl:text>&#10;</xsl:text>
	<xsl:element name="No-Match">
		<xsl:value-of select="Name/@First"/>
	</xsl:element>
</xsl:template>

</xsl:stylesheet>