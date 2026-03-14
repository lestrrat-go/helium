<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.whitespace.example.com/ns/various"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test that whitespace text nodes are not stripped from elements with simple 
content regardless of xsl:preserve-space or xsl:strip-space. -->

   <!-- We use several different tests to check the whitespace is still there,
   in order to exercise different paths in Saxon -->
   
   <!-- firstName has a simple type; familyName has a complex type with simple content -->

   <t:strip-space elements="firstName familyName"/>
   <!-- note, the child elements are not namespace-qualified -->

   <t:template match="/my:userNode">
      <t:variable name="copy">
         <out>
            <t:copy-of select="my:complexUserElem[2]"/>
         </out>
      </t:variable>
      <t:choose>
         <t:when test="empty($copy//firstName/text()[matches(.,'\s+')])">
            <a><t:copy-of select="$copy"/></a>
         </t:when>
         <t:when test="empty($copy//familyName/text()[matches(.,'\s+')])">
            <b><t:copy-of select="$copy"/></b>
         </t:when>
         <t:when test="empty(my:complexUserElem[2]//firstName/text()[matches(.,'\s+')])">
            <c><t:copy-of select="$copy"/></c>
         </t:when>
         <t:when test="string(my:complexUserElem[2]//firstName) = ''">
            <d><t:copy-of select="$copy"/></d>
         </t:when>
         <t:when test="not(data(my:complexUserElem[2]//firstName))">
            <e><t:copy-of select="$copy"/></e>
         </t:when>
         <t:when test="empty(my:complexUserElem[2]//familyName/text()[matches(.,'\s+')])">
            <f><t:copy-of select="$copy"/></f>
         </t:when>
         <t:when test="string(my:complexUserElem[2]//familyName) = ''">
            <g><t:copy-of select="$copy"/></g>
         </t:when>
         <t:when test="not(data(my:complexUserElem[2]//familyName))">
            <h><t:copy-of select="$copy"/></h>
         </t:when>
         <t:otherwise>
            <ok/>
         </t:otherwise>
      </t:choose>
   </t:template>
</t:transform>
