<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test case to verify that any document node within the result sequence is replaced 
  				by a sequence containing each of its children, in document order.-->

   
   <t:attribute-set name="attset">
      <t:attribute name="t1">1</t:attribute>
      <t:attribute name="t2" select="2"/>
      <t:attribute name="t3">
         <t:value-of select="3"/>
      </t:attribute>
   </t:attribute-set>
   
   <t:template match="/">
      <t:document>
         <out>
            <t:element name="e1" namespace="http://www.example.org/foo" use-attribute-sets="attset"/>

            <t:if test="true()">
               <t:comment>comment inside xsl:document</t:comment>
            </t:if>

            <t:choose>
               <t:when test="false()">
                  <t:processing-instruction name="Pro1">first pi inside xsl:document</t:processing-instruction>
               </t:when>
               <t:otherwise>
                  <t:processing-instruction name="Pro2">last pi inside xsl:document</t:processing-instruction>
               </t:otherwise>
            </t:choose>
            <t:copy/>

            <t:document>
               <test/>
            </t:document>

            <t:apply-templates/>

         </out>
      </t:document>
   </t:template>

   <t:template match="doc">
      <t:element name="e2"> inside child elements</t:element>
   </t:template>
</t:transform>
