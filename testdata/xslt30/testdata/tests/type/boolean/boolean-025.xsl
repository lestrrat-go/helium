<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
  <!-- PURPOSE: Test that 'false' converts to boolean "false" at XPath 2.0 
       (at XPath 1.0, it converts to true) -->
  <!-- Test changed Jan 2003, comparison of boolean to string no longer allowed -->
  <?same-as-1.0 no?>
<xsl:template match="/">
  <out result="true">
    <xsl:value-of select="false()=xs:boolean('false')" xmlns:xs="http://www.w3.org/2001/XMLSchema"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
