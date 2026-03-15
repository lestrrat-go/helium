<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="xml"
             include-content-type="yes"
             standalone="yes"
             encoding="UTF-8"/>

   <t:output cdata-section-elements="item1" indent="no"/>
   <!-- Purpose: Test of merging two unnamed output declarations with 
  				non-overlapping attributes. (output method is XML)-->

   <t:template match="/">
      <out>
         <item1>A &amp; B</item1>
         <item2>A &amp; B</item2>
      </out>
   </t:template>
</t:transform>
