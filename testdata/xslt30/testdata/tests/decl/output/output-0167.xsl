<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="xml"
             indent="no"
             encoding="UTF-8"
             normalization-form="NFC"/>
   <!-- Purpose: Test of XML output with @normalization-form=NFC.-->

   <t:template match="/">
      <html>
         <body>Á</body>
      </html>
   </t:template>
</t:transform>
