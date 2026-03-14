<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
  <!-- Purpose: Test IEEE 754 comparisons involving floating point numbers.
       Test of a predicate the "Less Than" predicate that holds
       true for the named comparison.  A "Less Than" relation
       yields a true value, a "Greater than" relation yields
       a false value, an "equal" relation yields a false value
       and if one of the operands is of the type "unordered",
       the relation yields a false value (IEEE 754 table 4). -->
  <!--  -->

<xsl:template match = "doc">
 <out><xsl:text>
   </xsl:text>
   <xsl:value-of select ="2.0 &lt; 2.1"/><xsl:text>
   </xsl:text>
   <xsl:value-of select ="2.0 > 2.1"/><xsl:text>
   </xsl:text>
   <xsl:value-of select ="2.0 = 2.1"/><xsl:text>
   </xsl:text>
   <xsl:value-of select ="2.0 &lt; NaN"/><xsl:text>
   </xsl:text>
 </out>
</xsl:template>

</xsl:stylesheet>
