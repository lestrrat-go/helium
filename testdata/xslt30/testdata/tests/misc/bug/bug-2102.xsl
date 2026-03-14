<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<?spec fo#func-boolean?>
<!-- BUG: Saxon 5.3.2/002 -->
<!-- DESCRIPTION: Conversion of a result-tree-fragment to a boolean is done incorrectly. The result
    should always be true; Saxon returns false if the result of converting the result tree
    fragment to a string is zero-length. -->

<!-- result of RTF to boolean conversion (should always be true) -->

<xsl:variable name="one"><xsl:text>true</xsl:text></xsl:variable>
<xsl:variable name="two"><xsl:text/></xsl:variable>

  <xsl:template match="/">
      <out><a one="{boolean($one)}" two="{boolean($two)}"/></out>
  </xsl:template>


</xsl:stylesheet>
