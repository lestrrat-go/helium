<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!--     Test case with doc($uri) where $uri is read from a node and represents 
  				relative URI resolved using the base-uri of the source document.
   -->

   <t:template match="/doc">
      <out>
         <t:copy-of select="doc(str1)//body"/>
      </out>
   </t:template>
</t:transform>
