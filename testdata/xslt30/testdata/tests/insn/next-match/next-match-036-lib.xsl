<?xml version="1.0" encoding="UTF-8"?>
<t:package xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0"
   name="http://xslt-tests/next-match-036-lib"
   expand-text="1" default-mode="m">
   <!-- Purpose: Current mode selected in a library package does not contain overriding template rules
   from the using package-->
   
   <t:mode name="m" visibility="public"/>

   <t:template match="doc">
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="tag">
      <tag>
         <t:next-match/>
      </tag>
   </t:template>

   <t:template match="bag">
      <bag>
         <t:next-match/>
      </bag>
   </t:template>
   
</t:package>
