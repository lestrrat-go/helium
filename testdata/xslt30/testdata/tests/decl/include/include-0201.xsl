<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#apply-imports?>
  <!-- PURPOSE: test that xsl:apply-imports invokes the built-in rule when there is
       no overridden template rule (bug Saxon 6.0.1/011) -->
  <xsl:template match="doc">
    <out><xsl:apply-imports/></out>
  </xsl:template>
 
</xsl:stylesheet>
