<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: test op:concatenate on nested sequences with path expressions -->


   <t:template match="doc">
      <out>
         <t:text>(</t:text>
         <t:value-of select="for $i in ((list[@id='list1'],                                     (list[@id='list2'],                                     (list[@id='list3']))),                                       list[@id='list4'])                          return $i/item"
                     separator=", "/>
         <t:text>)</t:text>
      </out>
   </t:template>
</t:transform>
