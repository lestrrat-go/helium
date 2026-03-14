<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output name="temp" method="text" indent="no"/>

   <t:output method="xhtml" indent="no" encoding="UTF-8"/>
   <!-- Purpose: Test with a named and unnamed output definition. 
  				Show that the unnamed output definition is used to 
  				serialize the implicit final result tree.-->

   <t:template match="/">
 	    <html>
         <body>
            <br/>
            <Option selected="selected"/>
         </body>
      </html>
   </t:template>
</t:transform>
