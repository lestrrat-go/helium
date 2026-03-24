<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://accum004/"  declared-modes="no"
    exclude-result-prefixes="xs f"
    version="3.0">
    
    <!-- Use accumulators to compute min, max, sum, avg, and count in parallel; streamed case -->
    
    <xsl:output method="xml" indent="yes"/>
    
   <xsl:accumulator name="count" as="xs:integer" initial-value="0" streamable="yes">
     <xsl:accumulator-rule match="transaction" select="$value + 1"/>
   </xsl:accumulator>
   
   <xsl:accumulator name="sum" as="xs:double" initial-value="0" streamable="yes">
     <xsl:accumulator-rule match="transaction" select="$value + @amount"/>
   </xsl:accumulator>
   
   <xsl:accumulator name="min" as="xs:double" initial-value="999999999999" streamable="yes">
     <xsl:accumulator-rule match="transaction" select="if (@amount &lt; $value) then @amount else $value"/>
   </xsl:accumulator>
   
   <xsl:accumulator name="max" as="xs:double" initial-value="-999999999999" streamable="yes">
     <xsl:accumulator-rule match="transaction" select="if (@amount &gt; $value) then @amount else $value"/>
   </xsl:accumulator>
 
  <xsl:mode streamable="yes" on-no-match="deep-skip" use-accumulators="count sum min max"/>
   <xsl:template match="/">
     <!-- cf. accumulator-009 which uses literal attributes; the streambility rules make a distinction,
          because the order of xsl:attribute instructions is assumed to be predictable -->
     <result>
       <xsl:attribute name="min" select="accumulator-after('min')"/>
       <xsl:attribute name="max" select="accumulator-after('max')"/>
       <xsl:attribute name="sum" select="accumulator-after('sum')"/>
       <xsl:attribute name="count" select="accumulator-after('count')"/>
       <xsl:attribute name="avg" select="round(accumulator-after('sum') div accumulator-after('count'), 2)"/>
     </result>
     
   </xsl:template>  
    
</xsl:package>