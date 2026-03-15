<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema">
                
<!-- A global variable is out of scope within its own declaration,
     even if it could be evaluated without circularity -->
     
<!-- See decision leading up to ACTION-ITEM-PRAGUE2013-005 -->     
           
  
  <xsl:template match="/">
     <out att="{$gcd(4,2)}"/>
     <xsl:message>*** Error not detected! ***</xsl:message>
  </xsl:template>
  
  <xsl:variable name="gcd" as="function(*)"
    select="
    function($x as xs:integer, $y as xs:integer) {
      if ($y eq 0)
      then abs($x)
      else $gcd($y,$x mod $y)
    }"
  />


  
</xsl:stylesheet>
