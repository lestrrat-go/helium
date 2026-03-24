<x xsl:version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xsl:exclude-result-prefixes="xs">
   <?spec fo#casting-to-numerics?>
 <xsl:value-of select="xs:decimal('123456789012345678') cast as xs:integer"/>
</x>
