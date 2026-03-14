<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
               xmlns:xs="http://www.w3.org/2001/XMLSchema"
               xmlns:f="function"
               exclude-result-prefixes="#all"  declared-modes="no"
               version="3.0">
               
   <!-- Use accumulator-rule containing a sequence constructor -->            

   <xsl:output method="xml" />
   
   <xsl:param name="streamable" static="yes"/>

   <xsl:mode _streamable="{$streamable}"  use-accumulators="#all"/>
   
   <xsl:global-context-item/>
  

   <xsl:accumulator name="item-cost" _streamable="{$streamable}" 
       as="xs:integer" initial-value="0">
       <xsl:accumulator-rule match="cost/text()">
         <xsl:sequence select="$value + xs:integer(.)"/>
       </xsl:accumulator-rule>  
   </xsl:accumulator>
   

   <xsl:template match="purchase-order">
       <Total-Cost>
           <items><xsl:value-of select="count(.//item)"/></items>
           <cost><xsl:value-of select="accumulator-after('item-cost')" /></cost>
       </Total-Cost>
   </xsl:template>

</xsl:package>