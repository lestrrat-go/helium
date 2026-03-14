<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- Testcase with doc(''). Verify the result is the stylesheet itself.-->

   <t:template match="/"><t:copy-of select="doc('')/t:transform/t:template"/></t:template>
   
</t:transform>
