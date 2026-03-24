<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:one="http://myoutputdefinition.example.org"
             xmlns:two="http://myoutputdefinition.example.org"
             xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="one two"
             version="2.0">

   <t:output name="one:temp" method="xhtml"/>

   <t:output name="two:temp" indent="no" encoding="UTF-8"/>
   <!-- Purpose: Show that merging of output declarations is done after expanding 
  				the QNames in @name. (use two declarations with the same local name 
  				but different prefixes bound to one and the same namespace)-->

   <t:template match="/">
      <t:result-document format="one:temp">
 	       <html>
            <body>
               <br/>
               <Option selected="selected"/>
            </body>
         </html>
      </t:result-document>
   </t:template>
</t:transform>
