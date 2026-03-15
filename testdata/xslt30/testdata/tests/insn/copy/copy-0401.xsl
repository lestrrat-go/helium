<?xml version="1.0"?> 

<?spec xslt#shallow-copy?>
<!-- BUG Saxon 5.5.1/010 -->
<!-- Test that contents of xsl:copy are ignored for an attribute, comment, text node, or PI -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:template match="/">
  <out>
    <xsl:for-each select="//node()">
       <xsl:copy>
          <xsl:for-each select="@*"><xsl:copy>content</xsl:copy></xsl:for-each>
          <xsl:text>contents of xsl:copy</xsl:text>
       </xsl:copy>
    </xsl:for-each>
  </out>     
  </xsl:template>
</xsl:stylesheet>
