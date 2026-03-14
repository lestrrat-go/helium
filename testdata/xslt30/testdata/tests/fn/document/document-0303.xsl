<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with doc(()). Result is the empty sequence-->

   <t:template match="/">
      <out>
         <a><t:copy-of select="doc(())"/></a>
         <b><t:copy-of select="doc(//str1)"/></b>
      </out>
   </t:template>
</t:transform>
