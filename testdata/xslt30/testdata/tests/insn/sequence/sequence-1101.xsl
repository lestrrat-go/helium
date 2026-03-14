<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
<!-- Purpose: Second example from spec. (modified) -->

   <xslt:variable name="prices" as="xs:decimal*">
      <xslt:for-each select="//product">
         <xslt:choose>
            <xslt:when test="@price">
               <xslt:sequence select="@price"/>
            </xslt:when>
            <xslt:otherwise>
               <xslt:sequence select="xs:decimal(@cost) * 1.5"/>
            </xslt:otherwise>
         </xslt:choose>
      </xslt:for-each>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="format-number(xs:double(avg($prices)), '#,###0.00')"/>
      </out>
   </xslt:template>
</xslt:transform>
