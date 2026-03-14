<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local/"
    exclude-result-prefixes="xs f"
    version="3.0">
    
    <!-- global variable cannot refer to itself, even if it would make sense -->
    
    <xsl:variable name="gcd" as="function(*)"
     select="
     function($x as xs:integer, $y as xs:integer) {
       if ($y eq 0)
       then abs($x)
       else $gcd($y,$x mod $y)
     }"/>
     
     <xsl:template name="xsl:initial-template">
       <out><xsl:sequence select="$gcd(2,2)"/></out>
     </xsl:template>

    
</xsl:stylesheet>