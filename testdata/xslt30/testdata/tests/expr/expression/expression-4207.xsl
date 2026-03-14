<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
  <!-- Purpose: Test IEEE 754 comparisons involving floating point numbers.
       Test of a predicate the "Greater Than" predicate that holds
       true for the named comparison.  A "Greater Than" relation
       yields a true value, a "Less than" relation yields
       a false value, an "equal" relation yields a false value
       and if one of the operands is of the type "unordered",
       the relation yields a false value (IEEE 754 table 4). -->
  <!--  -->

<xsl:template match = "doc">
 <out><xsl:text>
   </xsl:text>
   <xsl:value-of select ="2.1 > 2.0"/><xsl:text>
   </xsl:text>
   <xsl:value-of select ="2.1 &lt; 2.0"/><xsl:text>
   </xsl:text>
   <xsl:value-of select ="2.1 = 2.0"/><xsl:text>
   </xsl:text>
   <xsl:value-of select ="2.1 > NaN"/><xsl:text>
   </xsl:text>
 </out>
</xsl:template>

</xsl:stylesheet>
