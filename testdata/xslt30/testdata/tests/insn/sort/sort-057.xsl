<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Use sequence constructor in xsl:sort instead of select, refer to . in the constructor -->

   <xsl:output method="xml" encoding="UTF-8" indent="no"/>

   <xsl:variable name="var" select="(1,4,0,-2,005,300)" as="xs:anyAtomicType*"/>

   <xsl:template match="/">
      <out>
         <xsl:for-each select="$var">
            <xsl:sort data-type="number" order="descending">
               <xsl:value-of select="."/>
            </xsl:sort>
            <xsl:value-of select="."/>
            <xsl:value-of select="' | '"/>
         </xsl:for-each>
      </out>
   </xsl:template>
</xsl:transform>
