<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#id-value-comparisons?>
  <!-- PURPOSE: Test "fortran" operators applied to numbers (XPath 2.0) -->
  <?same-as-1.0 no?>
<xsl:template match="/">
<out>
  <true value="{1 eq 1.0}"/>
  <true value="{1 le 1.5}"/>
  <true value="{1.0e2 ne 1e3}"/>
  <true value="{10 lt 180.3}"/>
  <true value="{0.1 gt 0.01}"/>
  <true value="{1 ge 1.0}"/>
  <false value="{1 eq 2}"/>
  <false value="{1 gt 2}"/>
  <false value="{1 ne 1}"/>
  <false value="{1 ge 2}"/>
  <false value="{1 le 0}"/>
  <false value="{1 lt 0}"/>
</out>    
</xsl:template>
 
</xsl:stylesheet>
