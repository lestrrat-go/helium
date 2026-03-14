<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Test case with doc($uri) where $uri is the result from fn:resolve-ur and static-base-uri functions.-->

   <t:template match="/doc">
      <out>
         <t:value-of select="doc(resolve-uri(str1, static-base-uri()))//body"/>
      </out>
   </t:template>
</t:transform>
