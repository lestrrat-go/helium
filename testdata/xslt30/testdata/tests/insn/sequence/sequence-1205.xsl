<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Range expression - nested range expression -->

   <t:template match="/">
      <out>
         <t:value-of select="((1 to (xs:integer(doc/num) - 12)) to (1 to 1)) to (5 to (2+3))"
           xmlns:xs="http://www.w3.org/2001/XMLSchema"/>
      </out>
   </t:template>
</t:transform>
