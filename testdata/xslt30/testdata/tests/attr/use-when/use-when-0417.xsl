<?xml version="1.0" encoding="UTF-8"?>
<transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="xs"
    version="3.0">

    <!-- 
        Purpose: tests that use-when must not be in the xsl namespace on xsl elements
        Variant of previous, with xslt namespace set as the default namespace, has no effect on no-namespace of attribute
    -->


    <template match="/">
        <result xmlns="">
            <value-of select="'raises no error'" use-when="function-available('no-such-function')" xmlns="http://www.w3.org/1999/XSL/Transform" />
        </result>
    </template>
</transform>