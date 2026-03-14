<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="1.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of 1.0 stylesheet that has an unguarded instruction set to version 2.0,
    and using new 2.0 sub-element of that instruction, unguarded. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="var"
                  select="(-1,-50,4,0.5,0,-2,1,300,-17,-0.5)"
                  as="xs:double*"/>

   <xslt:template match="/">
      <out>
         <xslt:for-each select="$var">
            <xslt:sort data-type="number" order="descending" version="2.0">
               <xslt:choose>
                  <xslt:when test=". &lt; -10">0</xslt:when>
                  <xslt:when test=". &gt; -10 and . &lt; 0">
                     <xslt:value-of select="1 - (. div 10)"/>
                  </xslt:when>
                  <xslt:when test=".=0">1</xslt:when>
                  <xslt:otherwise>
                     <xslt:value-of select="1 + (. div 10)"/>
                  </xslt:otherwise>
               </xslt:choose>
            </xslt:sort>
            <xslt:value-of select="."/>
            <xslt:value-of select="'|'"/>
         </xslt:for-each>
      </out>
   </xslt:template>
</xslt:transform>
