<xsl:stylesheet 
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:f="http://local-functions.uri/"
   exclude-result-prefixes="xs f"
   version="2.0">
   
   <!-- show atomization of type-annotated element nodes -->
   
   <?spec xpath#id-typed-value?>
   <?spec xpath#id-function-calls?>
   
<xsl:import-schema namespace="http://ns.example.com/val03/" schema-location="validation-03.xsd"/>

<xsl:function name="f:total-numbers" as="xs:decimal*">
   <xsl:param name="input" as="xs:decimal*"/>
   <xsl:param name="total" as="xs:decimal"/>
   <xsl:if test="exists($input)">
      <xsl:variable name="x" as="xs:decimal"
                    select="$input[1] + $total"/>
      <xsl:sequence select="$x"/>
      <xsl:sequence select="f:total-numbers($input[position()!=1], $x)"/>
   </xsl:if>
</xsl:function>

<xsl:template match="/">
  <total values="{f:total-numbers(n:numbers, 0)}" 
           xmlns:n="http://ns.example.com/val03/"
           xsl:exclude-result-prefixes="n"/>
</xsl:template>

</xsl:stylesheet>
