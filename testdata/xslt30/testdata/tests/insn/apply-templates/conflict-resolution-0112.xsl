<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#conflict?>
<!-- PURPOSE: Testing Match of qualified name. Tests that "//*" has higher priority than "doc" or "foo" -->

<xsl:template match="doc">
   <wrong>
      <xsl:apply-templates select="foo"/>
   </wrong>
</xsl:template>
  
<xsl:template match="foo">
    <text>Match-of-qualified-name (incorrect)</text>
</xsl:template>

<xsl:template match="//*">
    <text>Match-of //* (correct)</text>
</xsl:template>

<xsl:template match="node()">
    <text>Match-of-node-type</text>
</xsl:template>

</xsl:stylesheet>
