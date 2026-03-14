<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#attribute-value-templates?>
  <!-- PURPOSE: Test two AVTs with literal element between them (based on example in the draft). -->
  <xsl:template match="photograph">
    <xsl:variable name="image-dir">/images</xsl:variable>
    <out src="{$image-dir}/{href}"/>
  </xsl:template>
 
</xsl:stylesheet>
