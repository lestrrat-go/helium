<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" >

<?spec xslt#patterns?>
<!-- BUG: 6.0.1/003 -->
<!-- DESCRIPTION: Keys on match="@*" not working -->

<xsl:key name="a" match="@*" use="name(..)"/>

<xsl:template match="/">
 <doc>
 <xsl:for-each select="key('a', 'stone')">
    <xsl:sort select="name(.)"/> <!-- to avoid dependency on attribute order -->
    <sss><xsl:value-of select="name(.)"/></sss>
 </xsl:for-each>
 </doc>
</xsl:template>


</xsl:stylesheet>
