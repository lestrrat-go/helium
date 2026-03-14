<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="text" encoding="UTF-8"/>

   <t:output name="temp"
             method="xml"
             include-content-type="yes"
             standalone="yes"
             encoding="UTF-8"/>

   <t:output name="temp" cdata-section-elements="item1" indent="no"/>
   <!-- Purpose: Test of merging two named output declarations with non-overlapping attributes.-->

   <t:template match="/">
      <t:result-document format="temp">
	        <out>
            <item1>A &amp; B</item1>
            <item2>A &amp; B</item2>
         </out>
      </t:result-document>
   </t:template>
</t:transform>
