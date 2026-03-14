<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test for conflict Resolution - nodetype. 
	 should see 1 conflict warning -->
	 
<xsl:template match="doc">
   <out>
      <xsl:apply-templates select="foo"/>
   </out>
</xsl:template>

<xsl:template match="*">
    <xsl:text>Match-of-wildcard</xsl:text>
</xsl:template>

<xsl:template match="node()">
    <xsl:text>Match-of-node-type</xsl:text>
</xsl:template>

</xsl:stylesheet>
