<?xml version="1.0" encoding="UTF-8"?>
<transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns="http://www.w3.org/1999/XSL/Transform"
    xmlns:out="urn:out"
    exclude-result-prefixes="xs out"
    version="3.0">

    <!-- 
        Purpose: Tests that use-when must be in the xsl namespace on LREs.
        Variant of previous, with xslt namespace set as the default namespace, has no effect on no-namespace of attribute, use-when has no effect
    -->


    <template match="/">
        <result xmlns="">
            <out:row select="'raises no error'" use-when="wrong///xpath" xmlns="http://www.w3.org/1999/XSL/Transform" />
        </result>
    </template>
</transform>