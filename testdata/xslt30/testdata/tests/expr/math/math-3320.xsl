<x xsl:version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xsl:exclude-result-prefixes="xs">
  <!-- Testing Round, xs:Decimal - round#2 not available with an XSLT 2.0 processor-->
   <?spec fo#func-round?>
    <out><xsl:value-of select="round(2.5, 1)"/></out>
</x>
