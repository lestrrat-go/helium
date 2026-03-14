<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test run-time formatting -->
  <xsl:template match="doc">
    <out>
      <xsl:variable name="format">00000000000001</xsl:variable>
      <xsl:variable name="sep">/</xsl:variable>
      <xsl:variable name="gp">2</xsl:variable>
      <xsl:number value="1000000" format="{$format}"
            grouping-separator="{$sep}" grouping-size="{$gp}"/>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
