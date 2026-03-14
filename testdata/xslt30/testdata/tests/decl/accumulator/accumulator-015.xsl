<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
               xmlns:xs="http://www.w3.org/2001/XMLSchema"
               xmlns:f="function"  declared-modes="no"
               exclude-result-prefixes="#all"
               version="3.0">

   <xsl:output method="xml" />

   <xsl:mode streamable="yes"  use-accumulators="item-cost"/>

   <xsl:accumulator name="item-cost" streamable="yes" 
       as="xs:integer" 
       initial-value="0">
       <xsl:accumulator-rule match="cost/text()" select="$value + xs:integer(.)"/>
   </xsl:accumulator>
   
   
   <!-- The void xsl:for-each in the template below is designed to sow confusion. Although it does nothing useful,
     it still serves to separate the pre-descent and post-descent phases of the sequence constructor. -->

   <xsl:template match="purchase-order">
       <Total-Cost>
           <xsl:for-each select="item">
               <xsl:for-each select="cost"/>
           </xsl:for-each>
           <xsl:value-of select="accumulator-after('item-cost')" />
       </Total-Cost>
   </xsl:template>

</xsl:package>