<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="html" indent="no"/>
   <!-- Purpose: Test HTML output method with illegal content inside 
				the 'script' tag. Example from spec. The escaping must be done manually. -->

   <t:template match="/">
      <html>
         <head>
            <script type="text/javascript">document.write ("&lt;EM&gt;This will work&lt;\/EM&gt;")</script>
         </head>
         <body/>
      </html>
   </t:template>
</t:transform>
