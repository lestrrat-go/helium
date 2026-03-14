<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.whitespace.example.com/ns/various"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" 
   exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test that whitespace text nodes are not stripped from an element
        if there is an ancestor element that was validated against a type that includes
        an assertion. -->

   <!-- We use several different tests to check the whitespace is still there,
   in order to exercise different paths in Saxon -->
   
   <!-- firstName has a simple type; familyName has a complex type with simple content -->

   <t:strip-space elements="p"/>
   


   <t:template match="/">
      <t:variable name="copy">
         <out>
            <t:copy-of select="doc"/>
         </out>
      </t:variable>
      <t:choose>
         <t:when test="empty($copy//p/text()[matches(.,'\s{4}')])">
            <a><t:copy-of select="$copy"/></a>
         </t:when>
         <t:when test="empty(doc//p/text()[matches(.,'\s+')])">
            <c><t:copy-of select="$copy"/></c>
         </t:when>
         <t:when test="string(doc//p) = ''">
            <d><t:copy-of select="$copy"/></d>
         </t:when>
         <t:when test="not(data(doc//p))">
            <e><t:copy-of select="$copy"/></e>
         </t:when>
         <t:otherwise>
            <ok/>
         </t:otherwise>
      </t:choose>
   </t:template>
</t:transform>
