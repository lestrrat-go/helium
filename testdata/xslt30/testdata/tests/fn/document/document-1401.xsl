<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with doc($uri) where the argument is a local HTML file.-->

   <t:output method="xml"/>

   <t:template match="/">
      <t:copy-of select="doc('doc17.html')"/>
   </t:template>
</t:transform>
