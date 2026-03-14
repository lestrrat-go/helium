<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#lexical-structure?>
    <!-- Purpose: * after ( must not be treated as operator, but * after ) is,
     and being tokenized means following * is not (because it follows an operator) -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="(* - 4)**"/>
  </out>
</xsl:template>

</xsl:stylesheet>
