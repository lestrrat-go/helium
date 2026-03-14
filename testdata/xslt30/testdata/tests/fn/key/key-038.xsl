<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#keys?>
    <!-- Purpose: Test complex key()// filtered by a predicate in predicate, 
       in a match pattern. -->


<xsl:key name="Info" match="Level1 | Level2" use="@ID"/>

<xsl:template match="/">
 <out>
	<xsl:apply-templates select="//Level3"/>
 </out>
</xsl:template>

<xsl:template match="key('Info','id1')//Level3[Name[starts-with(@First,'J')]]">
	<xsl:text>&#10;</xsl:text>
	<xsl:element name="J-NAME">
		<xsl:value-of select="Name/@First"/>
	</xsl:element>
</xsl:template>


<xsl:template match="*">
	<xsl:text>&#10;</xsl:text>
	<xsl:element name="Name">
		<xsl:value-of select="Name/@First"/>
	</xsl:element>
</xsl:template>

</xsl:stylesheet>