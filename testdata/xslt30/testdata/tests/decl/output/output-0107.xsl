<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml" indent="no"/>
   <!-- Purpose: Test for SCRIPT handling with XHTML output method. -->

   <t:template match="/">
      <html>
         <head>
            <script>
        
        document.write("&lt;p&gt;hi oren&lt;/p&gt;");
        if((2 &lt; 5) &amp; (6 &lt; 3))
        {
          ...
        }
        
      </script>
         </head>
         <body/>
      </html>
   </t:template>
</t:transform>
