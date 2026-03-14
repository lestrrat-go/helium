<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test subsequence() second and third arguments being cast to double. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <first>
            <t:value-of select="subsequence(('a','b','c','d','e','f','g','h','i'),'4')"
               separator=", "/>
         </first>
         <second>
            <t:value-of select="subsequence(('a','b','c','d'),'1.000',3)" separator=", "/>
         </second>
         <third>
            <t:value-of select="subsequence(('a','b','c','d','e','f','g','h','i'),doc/c/@n,'4')"
               separator=", "/>
         </third>
         <fourth>
            <t:value-of select="subsequence(('a','b','c','d','e','f','g','h','i'),'2.00001','4')"
               separator=", "/>
         </fourth>
      </out>
   </t:template>
</t:transform>
