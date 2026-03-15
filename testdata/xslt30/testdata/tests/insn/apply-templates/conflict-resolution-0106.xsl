<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#conflict?>
<!-- Test for conflict Resolution - nodetest override of default priority.
     Also, node selected is attribute instead of element.
     No conflict warnings should be seen. -->

<xsl:template match="doc" priority="10">
   <out>
     <xsl:apply-templates select="foo/@test"/>
   </out>
</xsl:template>

<xsl:template match="foo">
    <xsl:text>Match-of-qualified-name</xsl:text>
</xsl:template>

<xsl:template match="*">
    <xsl:text>Match-of-wildcard</xsl:text>
</xsl:template>

<xsl:template match="node()" priority="1">
    <xsl:text>Match-of-node-type</xsl:text>
</xsl:template>

</xsl:stylesheet>
