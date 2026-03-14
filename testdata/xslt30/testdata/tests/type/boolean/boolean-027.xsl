<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#id-value-comparisons?>
  <!-- PURPOSE: Test "fortran" operators applied to strings (XPath 2.0) -->
  <?same-as-1.0 no?>
<xsl:template match="/">
<out>
  <t01 value="{'1' eq '1'}"/>
  <t02 value="{'1' le '1.5'}"/>
  <t03 value="{'1.0e2' ne '1e3'}"/>
  <f04 value="{'20' lt '180.3'}"/>
  <t05 value="{'0.1' gt '0.01'}"/>
  <f06 value="{'1' ge '1.0'}"/>
  <f07 value="{'1' eq '10'}"/>
  <f08 value="{'1' gt '2'}"/>
  <f09 value="{'1' ne '1'}"/>
  <f10 value="{'1' ge '2'}"/>
  <f11 value="{'1' le '0'}"/>
  <f12 value="{'1' lt '0'}"/>
</out>    
</xsl:template>
 
</xsl:stylesheet>
