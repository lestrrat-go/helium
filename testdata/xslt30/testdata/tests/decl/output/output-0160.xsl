<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="html" indent="no"/>
   <!-- Purpose: Test that whitespace is not added or removed adjacent to elements 
  				ins and del if they do not contain element children; indent=no with HTML output method.-->

   <t:template match="/">
      <html>
         <head>This is the head</head>
         <body>
            <p>a dozen is <del>
    			20	</del>
               <ins>     1 2 </ins> pieces</p>
         </body>
      </html>
   </t:template>
</t:transform>
