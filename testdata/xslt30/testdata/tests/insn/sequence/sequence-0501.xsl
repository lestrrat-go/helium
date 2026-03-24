<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: apply positional predicate to sequence. -->


   <t:template match="doc">
      <out>
         <t:text>(</t:text>
         <t:for-each select="(for $i in (list[@id='list1']/item,                             (list[@id='list2']/item, list[@id='list3']/item),                             list[@id='list4']/item)                           return $i)                           [position() &gt; 3 and position() &lt; 8]">
            <t:copy-of select="text()"/>
            <t:if test="not(last()=position())">
               <t:text>|</t:text>
            </t:if>
         </t:for-each>
         <t:text>)</t:text>
      </out>
   </t:template>
</t:transform>
