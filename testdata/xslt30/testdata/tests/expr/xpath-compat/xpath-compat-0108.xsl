<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" version="2.0" exclude-result-prefixes="xs">
   <!-- Purpose: Test the div operator on two type-annotated integers. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="top" select="12" as="xs:integer"/>

   <xslt:variable name="even" select="4" as="xs:integer"/>

   <xslt:variable name="odd" select="5" as="xs:integer"/>

   <xslt:template match="/" version="1.0">
      <out>
         <xslt:choose>
            <xslt:when test="($top div $odd) instance of xs:double">
               <DivOdd>double</DivOdd>
            </xslt:when>
            <xslt:when test="($top div $odd) instance of xs:decimal">
               <DivOdd>decimal</DivOdd>
            </xslt:when>
            <xslt:when test="($top div $odd) instance of xs:integer">
               <DivOdd>integer</DivOdd>
            </xslt:when>
            <xslt:otherwise>
               <DivOdd>ERROR: other type</DivOdd>
            </xslt:otherwise>
         </xslt:choose>
         <xslt:choose>
            <xslt:when test="($top div $even) instance of xs:double">
               <DivOdd>double</DivOdd>
            </xslt:when>
            <xslt:when test="($top div $even) instance of xs:decimal">
               <DivOdd>decimal</DivOdd>
            </xslt:when>
            <xslt:when test="($top div $even) instance of xs:integer">
               <DivOdd>integer</DivOdd>
            </xslt:when>
            <xslt:otherwise>
               <DivOdd>ERROR: other type</DivOdd>
            </xslt:otherwise>
         </xslt:choose>
      </out>
   </xslt:template>
</xslt:transform>
