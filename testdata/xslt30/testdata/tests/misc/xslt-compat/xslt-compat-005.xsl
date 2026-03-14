<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Use a path expression that selects more than one node in an attribute value template. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <sometag attributevalue="{//item[@name='text']}"/>
      </out>
   </t:template>
</t:transform>
