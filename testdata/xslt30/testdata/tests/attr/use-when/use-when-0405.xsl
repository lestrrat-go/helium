<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to confirm that whitespace stripping happens before element exclusion.-->

   <t:template match="/">
      <out>
         <in t:use-when="false()"/>xxx
      </out>     
   </t:template>
</t:transform>
