<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <xsl:param name="o" select="'1406'"/>

   <!-- Purpose: Test serialization with parameter document -->

   <xsl:template name="xsl:initial-template">
      <xsl:result-document parameter-document="result-document-{$o}-params.xml" build-tree="no">
         <xsl:map>
            <xsl:map-entry key="'a'" select="'AAA'"/>
            <xsl:map-entry key="'b'" select="'BBB'"/>
            <xsl:map-entry key="'c'" select="'CCC'"/>
            <xsl:map-entry key="'d'" select="'DDD'"/>
            <xsl:map-entry key="'e'" select="'EEE'"/>
            <xsl:map-entry key="'f'" select="'FFF'"/>
            <xsl:map-entry key="'g'" select="'GGG'"/>
         </xsl:map>
      </xsl:result-document>
   </xsl:template>
</xsl:transform>
