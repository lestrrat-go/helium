<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: test op:concatenate of nested sequences with path expressions, for clause, and for-each -->

   <t:template match="doc">
      <out>
         <t:text>(</t:text>
         <t:for-each select="for $i in ((list[@id='list1'],                                     (list[@id='list2'],                                     (list[@id='list3']))),                                       list[@id='list4'])                          return $i">
            <t:text>[</t:text>
            <t:for-each select="item">
               <t:copy-of select="text()"/>
               <t:if test="not(last()=position())">
                  <t:text>|</t:text>
               </t:if>
            </t:for-each>
            <t:text>]</t:text>
         </t:for-each>
         <t:text>)</t:text>
      </out>
   </t:template>
</t:transform>
