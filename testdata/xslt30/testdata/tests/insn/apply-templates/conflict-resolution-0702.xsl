<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#unprefixed-qnames?>
<!-- Testing xsl:xpath-default-namespace -->
<xsl:template match="u:doc" xmlns:u="http://some.uri/">
   <out xsl:xpath-default-namespace="http://some.uri/">
      <xsl:apply-templates select="foo"/>
   </out>
</xsl:template>
  
<xsl:template match="u:foo" xmlns:u="http://some.uri/">
    <xsl:text>Match-of-qualified-name</xsl:text>
</xsl:template>

<xsl:template match="*">
    <xsl:text>Match-of-wildcard</xsl:text>
</xsl:template>

<xsl:template match="node()">
    <xsl:text>Match-of-node-type</xsl:text>
</xsl:template>

</xsl:stylesheet>
