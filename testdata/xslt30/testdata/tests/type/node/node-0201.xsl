<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#node-tests?>
    <!-- Purpose: Test for comment() node test. -->

<xsl:template match="/doc">
  <out>      
    <xsl:apply-templates select="./comment()"/>
  </out>
</xsl:template>

<xsl:template match="comment()">
  <xsl:text>Found-comment</xsl:text>
  <xsl:copy/>
</xsl:template>

</xsl:stylesheet>
