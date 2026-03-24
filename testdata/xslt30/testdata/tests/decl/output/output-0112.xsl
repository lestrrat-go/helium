<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml"
             doctype-public="-//BOAG//DTD Websites V1.3//EN"
             indent="no"
             encoding="UTF-8"/>
   <!-- Purpose: Test  XHTML output with @doctype-public only. Show that attribute is ignored.-->

   <t:template match="/">
      <html/>
   </t:template>
</t:transform>
