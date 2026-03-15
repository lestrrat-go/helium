<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="text"
             indent="no"
             encoding="UTF-8"
             normalization-form="none"/>
   <!-- Purpose: Test of TEXT output with @normalization-form=none.-->

   <t:template match="/">
      <html>
         <body>Á</body>
      </html>
   </t:template>
</t:transform>
