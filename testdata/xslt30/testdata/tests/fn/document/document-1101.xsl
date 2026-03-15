<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Try sorting and numbering nodes from two other documents retrieved by doc().-->

   <t:template match="doc">
      <out>
         <t:value-of select="count(doc(a[1])//body)"/>
         <t:text> * </t:text>
         <t:value-of select="count(doc(a[3])//body)"/>
         <t:text> body nodes:
</t:text>
         <t:apply-templates select="doc(a[1])//body">
            <t:sort select="substring-after(.,'-')"/>
         </t:apply-templates>
         <t:text> *
</t:text>
         <t:apply-templates select="doc(a[3])//body">
            <t:sort select="substring-after(.,'-')"/>
         </t:apply-templates>
      </out>
   </t:template>

   <t:template match="body">
      <t:value-of select="position()"/>
      <t:text>. </t:text>
      <t:value-of select="."/>
      <t:text>
</t:text>
   </t:template>
</t:transform>
