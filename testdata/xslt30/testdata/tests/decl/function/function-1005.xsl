<?xml version="1.0"?>
<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:math="http://exslt.org/math"
                exclude-result-prefixes="math xs">
                
<!-- recursive stylesheet functions used to do numeric computation -->

<xsl:template match="/">
  <out><xsl:sequence select="format-number(math:squareRoot(10), '#.0000')" /></out>
</xsl:template>

<xsl:function name="math:squareRoot" as="xs:double">
  <xsl:param name="number" as="xs:double" /> 
  <xsl:sequence select="math:squareRoot($number, 4)" />
</xsl:function>

<xsl:function name="math:squareRoot" as="xs:double">
  <xsl:param name="number" as="xs:double" /> 
  <xsl:param name="precision" as="xs:integer" />
  <xsl:sequence select="math:_squareRoot($number, $precision, 1)" />
</xsl:function>

<xsl:function name="math:_squareRoot" as="xs:double">
  <xsl:param name="number" as="xs:double" /> 
  <xsl:param name="precision" as="xs:integer" />
  <xsl:param name="estimate" as="xs:double" />
  <xsl:variable name="nextEstimate" 
    select="$estimate + (($number - $estimate * $estimate) div 
                         (2 * $estimate))" />
  <xsl:variable name="nextEstimate" as="xs:double"
    select="round-half-to-even($nextEstimate, $precision)" />
  <xsl:sequence
    select="if ($estimate = $nextEstimate) then $estimate
            else math:_squareRoot($number, $precision, $nextEstimate)" />
</xsl:function>

</xsl:stylesheet>