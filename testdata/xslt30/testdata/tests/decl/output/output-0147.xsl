<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml"
             indent="no"
             encoding="UTF-8"
             normalization-form="none"/>
   <!-- Purpose: Test of XTHML output with @normalization-form=none. Note,the body element contains x41 xCC x81,
      representing the two codepoints 65 769 ('A' followed by the combining character x301) -->

   <t:template match="/">
      <html>
         <body>Á</body>
      </html>
   </t:template>
</t:transform>
