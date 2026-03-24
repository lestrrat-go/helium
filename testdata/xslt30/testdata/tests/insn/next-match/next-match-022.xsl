<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test that t:next-match stays in same mode as its caller.-->

   <t:template match="doc">
      <out>
         <t:apply-templates mode="yes"/>
      </out>
   </t:template>

   <t:template match="tag" mode="yes">
      <div style="border: solid green">
         <t:apply-imports/>
      </div>
   </t:template>

   <t:template match="tag">
      <div style="border: solid red">
         <t:apply-imports/>
      </div>
   </t:template>

   <t:template match="tag" mode="no">
      <bad>
         <t:apply-templates/>
      </bad>
   </t:template>
   
</t:transform>
