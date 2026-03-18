<?xml version="1.0" encoding="UTF-8"?>
<scm:schema xmlns:scm="http://ns.saxonica.com/schema-component-model"
            generatedAt="2017-02-04T16:45:57.789Z"
            xsdVersion="1.1">
   <scm:simpleType id="C0"
                   name="nametests"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="list"
                   itemType="C1"/>
   <scm:simpleType id="C2"
                   name="yes-or-no-or-omit"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="no"/>
      <scm:enumeration value="yes"/>
      <scm:enumeration value="false"/>
      <scm:enumeration value="0"/>
      <scm:enumeration value="1"/>
      <scm:enumeration value="true"/>
      <scm:enumeration value="omit"/>
   </scm:simpleType>
   <scm:simpleType id="C3"
                   name="simpleDerivationSet"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C4 C5"/>
   <scm:simpleType id="C6"
                   name="on-no-match-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="shallow-skip"/>
      <scm:enumeration value="text-only-copy"/>
      <scm:enumeration value="fail"/>
      <scm:enumeration value="deep-skip"/>
      <scm:enumeration value="deep-copy"/>
      <scm:enumeration value="shallow-copy"/>
   </scm:simpleType>
   <scm:complexType id="C7"
                    name="assertion"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C9"/>
      <scm:attributeUse required="false" inheritable="false" ref="C10"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C14"
                    name="all"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C15"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C16" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C17" default="1"/>
      <scm:attributeWildcard ref="C18"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C19"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C20"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C21"/>
               </scm:choice>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C21" to="1"/>
            <scm:edge term="C13" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C21" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C21" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C22"
                   name="prefixes"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="list"
                   itemType="#NCName"/>
   <scm:complexType id="C23"
                    name="generic-element-type"
                    targetNamespace="http://www.w3.org/1999/XSL/Transform"
                    base="#anyType"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C41"
                    name="restrictionType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="true" inheritable="false" ref="C42"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
                     <scm:choice>
                        <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
                           <scm:sequence>
                              <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C43"/>
                              <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C44"/>
                           </scm:sequence>
                        </scm:modelGroupParticle>
                        <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C45"/>
                     </scm:choice>
                  </scm:modelGroupParticle>
                  <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C46"/>
                  <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C47"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C48" to="3"/>
            <scm:edge term="C49" to="3"/>
            <scm:edge term="C50" to="3"/>
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C13" to="7"/>
            <scm:edge term="C43" to="4"/>
            <scm:edge term="C52" to="3"/>
            <scm:edge term="C53" to="3"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C55" to="3"/>
            <scm:edge term="C56" to="3"/>
            <scm:edge term="C57" to="3"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C59" to="3"/>
            <scm:edge term="C60" to="5"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C62" to="3"/>
            <scm:edge term="C63" to="8"/>
            <scm:edge term="C64" to="3"/>
            <scm:edge term="C65" to="3"/>
            <scm:edge term="C66" to="6"/>
            <scm:edge term="C67" to="3"/>
            <scm:edge term="C68" to="3"/>
            <scm:edge term="C69" to="3"/>
            <scm:edge term="C70" to="5"/>
            <scm:edge term="C71" to="2"/>
            <scm:edge term="C72" to="3"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C63" to="8"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="5"/>
            <scm:edge term="C70" to="5"/>
            <scm:edge term="C63" to="8"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C48" to="3"/>
            <scm:edge term="C49" to="3"/>
            <scm:edge term="C50" to="3"/>
            <scm:edge term="C52" to="3"/>
            <scm:edge term="C53" to="3"/>
            <scm:edge term="C55" to="3"/>
            <scm:edge term="C56" to="3"/>
            <scm:edge term="C57" to="3"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C59" to="3"/>
            <scm:edge term="C60" to="5"/>
            <scm:edge term="C62" to="3"/>
            <scm:edge term="C63" to="8"/>
            <scm:edge term="C64" to="3"/>
            <scm:edge term="C65" to="3"/>
            <scm:edge term="C67" to="3"/>
            <scm:edge term="C68" to="3"/>
            <scm:edge term="C69" to="3"/>
            <scm:edge term="C70" to="5"/>
            <scm:edge term="C72" to="3"/>
         </scm:state>
         <scm:state nr="4">
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="5" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="5"/>
            <scm:edge term="C70" to="5"/>
            <scm:edge term="C63" to="8"/>
         </scm:state>
         <scm:state nr="6" final="true">
            <scm:edge term="C48" to="3"/>
            <scm:edge term="C49" to="3"/>
            <scm:edge term="C50" to="3"/>
            <scm:edge term="C52" to="3"/>
            <scm:edge term="C53" to="3"/>
            <scm:edge term="C55" to="3"/>
            <scm:edge term="C56" to="3"/>
            <scm:edge term="C57" to="3"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C59" to="3"/>
            <scm:edge term="C60" to="5"/>
            <scm:edge term="C62" to="3"/>
            <scm:edge term="C63" to="8"/>
            <scm:edge term="C64" to="3"/>
            <scm:edge term="C65" to="3"/>
            <scm:edge term="C67" to="3"/>
            <scm:edge term="C68" to="3"/>
            <scm:edge term="C69" to="3"/>
            <scm:edge term="C70" to="5"/>
            <scm:edge term="C72" to="3"/>
         </scm:state>
         <scm:state nr="7" final="true">
            <scm:edge term="C48" to="3"/>
            <scm:edge term="C49" to="3"/>
            <scm:edge term="C50" to="3"/>
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C43" to="4"/>
            <scm:edge term="C52" to="3"/>
            <scm:edge term="C53" to="3"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C55" to="3"/>
            <scm:edge term="C56" to="3"/>
            <scm:edge term="C57" to="3"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C59" to="3"/>
            <scm:edge term="C60" to="5"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C62" to="3"/>
            <scm:edge term="C63" to="8"/>
            <scm:edge term="C64" to="3"/>
            <scm:edge term="C65" to="3"/>
            <scm:edge term="C66" to="6"/>
            <scm:edge term="C67" to="3"/>
            <scm:edge term="C68" to="3"/>
            <scm:edge term="C69" to="3"/>
            <scm:edge term="C70" to="5"/>
            <scm:edge term="C71" to="2"/>
            <scm:edge term="C72" to="3"/>
         </scm:state>
         <scm:state nr="8" final="true">
            <scm:edge term="C63" to="8"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C73"
                    name="element-only-versioned-element-type"
                    targetNamespace="http://www.w3.org/1999/XSL/Transform"
                    base="C74"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C78"
                   name="sequence-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:pattern value=".+"/>
   </scm:simpleType>
   <scm:simpleType id="C79"
                   name="input-type-annotations-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="unspecified"/>
      <scm:enumeration value="preserve"/>
      <scm:enumeration value="strip"/>
   </scm:simpleType>
   <scm:complexType id="C80"
                    name="attributeGroupRef"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C81"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C82"/>
      <scm:attributeWildcard ref="C83"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C84"
                   name="zero-digit"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="C85"
                   variety="atomic"
                   primitiveType="#string">
      <scm:pattern value="\p{Nd}"/>
      <scm:assert>
         <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                        xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                        xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                        test="matches(string-join(codepoints-to-string(for $i in 0 to 9                            return string-to-codepoints($value) + $i), ''), '\p{Nd}{10}')"
                        defaultNamespace=""
                        xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      </scm:assert>
      <scm:length value="1"/>
   </scm:simpleType>
   <scm:complexType id="C86"
                    name="altType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C87"/>
      <scm:attributeUse required="false" inheritable="false" ref="C88"/>
      <scm:attributeUse required="false" inheritable="false" ref="C89"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C66"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C90"/>
               </scm:choice>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C90" to="1"/>
            <scm:edge term="C13" to="2"/>
            <scm:edge term="C66" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2" final="true">
            <scm:edge term="C90" to="1"/>
            <scm:edge term="C66" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C91"
                   name="prefix-list"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="list"
                   itemType="C92"/>
   <scm:simpleType id="C93"
                   name="validation-strip-or-preserve"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="C94"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="preserve"/>
      <scm:enumeration value="strip"/>
   </scm:simpleType>
   <scm:complexType id="C95"
                    name="sequence-constructor-or-select"
                    targetNamespace="http://www.w3.org/1999/XSL/Transform"
                    base="C96"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:simpleType id="C146"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="#default"/>
      <scm:enumeration value="#unnamed"/>
   </scm:simpleType>
   <scm:simpleType id="C147"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="#all"/>
   </scm:simpleType>
   <scm:simpleType id="C148"
                   name="streamability-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C149 C150"/>
   <scm:complexType id="C151"
                    name="anyType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="#anyType"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="mixed">
      <scm:attributeWildcard ref="C152"/>
      <scm:elementWildcard minOccurs="0" maxOccurs="unbounded" ref="C153"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C153" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C153" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C154"
                   name="on-multiple-match-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="use-last"/>
      <scm:enumeration value="fail"/>
   </scm:simpleType>
   <scm:simpleType id="C155"
                   name="pattern"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="C156"
                   variety="atomic"
                   primitiveType="#string">
      <scm:pattern value=".+"/>
   </scm:simpleType>
   <scm:simpleType id="C157"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="#anyURI C158"/>
   <scm:simpleType id="C159"
                   name="zero-length-string"
                   targetNamespace="http://saxon.sf.net/"
                   base="#string"
                   variety="atomic"
                   primitiveType="#string">
      <scm:length value="0"/>
   </scm:simpleType>
   <scm:simpleType id="C160"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="#all"/>
   </scm:simpleType>
   <scm:simpleType id="C161"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string"/>
   <scm:complexType id="C162"
                    name="text-element-type"
                    targetNamespace="http://www.w3.org/1999/XSL/Transform"
                    base="C163"
                    derivationMethod="extension"
                    abstract="false"
                    variety="simple"
                    simpleType="C164">
      <scm:attributeUse required="false" inheritable="false" ref="C165" default="no"/>
      <scm:attributeUse required="false" inheritable="false" ref="C166"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C167"/>
   </scm:complexType>
   <scm:complexType id="C168"
                    name="namedAttributeGroup"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C81"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C169"/>
      <scm:attributeWildcard ref="C170"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C46"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C13" to="3"/>
            <scm:edge term="C70" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C70" to="2"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C70" to="2"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C171"
                    name="realGroup"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C172"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C173" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C174" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C175"/>
      <scm:attributeUse required="false" inheritable="false" ref="C176"/>
      <scm:attributeWildcard ref="C177"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C71"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C51"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C61"/>
               </scm:choice>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
            <scm:edge term="C71" to="2"/>
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C61" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C71" to="2"/>
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C61" to="2"/>
         </scm:state>
         <scm:state nr="2" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C178"
                    name="extensionType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="true" inheritable="false" ref="C179"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C43"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="1" ref="C44"/>
                  <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C46"/>
                  <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C47"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C13" to="5"/>
            <scm:edge term="C43" to="4"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C63" to="6"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C63" to="6"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="6"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="6"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C71" to="2"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="6"/>
         </scm:state>
         <scm:state nr="5" final="true">
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C43" to="4"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C63" to="6"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="6" final="true">
            <scm:edge term="C63" to="6"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C180"
                   name="accumulator-names"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C181 C182"/>
   <scm:simpleType id="C183"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="##targetNamespace"/>
      <scm:enumeration value="##local"/>
      <scm:enumeration value="##defaultNamespace"/>
   </scm:simpleType>
   <scm:simpleType id="C184"
                   name="visibility-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="private"/>
      <scm:enumeration value="hidden"/>
      <scm:enumeration value="abstract"/>
      <scm:enumeration value="public"/>
      <scm:enumeration value="final"/>
   </scm:simpleType>
   <scm:simpleType id="C185"
                   name="default-mode-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C186 C187"/>
   <scm:simpleType id="C188"
                   name="_langType"
                   targetNamespace="http://www.w3.org/XML/1998/namespace"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="#language C159"/>
   <scm:simpleType id="C189"
                   base="#NMTOKEN"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="replace"/>
      <scm:enumeration value="preserve"/>
      <scm:enumeration value="collapse"/>
   </scm:simpleType>
   <scm:complexType id="C190"
                    name="numFacet"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C191"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C192"/>
      <scm:attributeUse required="false" inheritable="false" ref="C193" default="false"/>
      <scm:attributeWildcard ref="C194"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C195"
                    name="simpleRestrictionType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C41"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C42"/>
      <scm:attributeWildcard ref="C196"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
               <scm:choice>
                  <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C45"/>
               </scm:choice>
            </scm:modelGroupParticle>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C46"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C47"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C48" to="3"/>
            <scm:edge term="C49" to="3"/>
            <scm:edge term="C50" to="3"/>
            <scm:edge term="C13" to="5"/>
            <scm:edge term="C52" to="3"/>
            <scm:edge term="C53" to="3"/>
            <scm:edge term="C55" to="3"/>
            <scm:edge term="C56" to="3"/>
            <scm:edge term="C57" to="3"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C59" to="3"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C62" to="3"/>
            <scm:edge term="C63" to="6"/>
            <scm:edge term="C64" to="3"/>
            <scm:edge term="C65" to="3"/>
            <scm:edge term="C66" to="4"/>
            <scm:edge term="C67" to="3"/>
            <scm:edge term="C68" to="3"/>
            <scm:edge term="C69" to="3"/>
            <scm:edge term="C70" to="2"/>
            <scm:edge term="C72" to="3"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C63" to="6"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C70" to="2"/>
            <scm:edge term="C63" to="6"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C48" to="3"/>
            <scm:edge term="C49" to="3"/>
            <scm:edge term="C50" to="3"/>
            <scm:edge term="C52" to="3"/>
            <scm:edge term="C53" to="3"/>
            <scm:edge term="C55" to="3"/>
            <scm:edge term="C56" to="3"/>
            <scm:edge term="C57" to="3"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C59" to="3"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C62" to="3"/>
            <scm:edge term="C63" to="6"/>
            <scm:edge term="C64" to="3"/>
            <scm:edge term="C65" to="3"/>
            <scm:edge term="C67" to="3"/>
            <scm:edge term="C68" to="3"/>
            <scm:edge term="C69" to="3"/>
            <scm:edge term="C70" to="2"/>
            <scm:edge term="C72" to="3"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C48" to="3"/>
            <scm:edge term="C49" to="3"/>
            <scm:edge term="C50" to="3"/>
            <scm:edge term="C52" to="3"/>
            <scm:edge term="C53" to="3"/>
            <scm:edge term="C55" to="3"/>
            <scm:edge term="C56" to="3"/>
            <scm:edge term="C57" to="3"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C59" to="3"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C62" to="3"/>
            <scm:edge term="C63" to="6"/>
            <scm:edge term="C64" to="3"/>
            <scm:edge term="C65" to="3"/>
            <scm:edge term="C67" to="3"/>
            <scm:edge term="C68" to="3"/>
            <scm:edge term="C69" to="3"/>
            <scm:edge term="C70" to="2"/>
            <scm:edge term="C72" to="3"/>
         </scm:state>
         <scm:state nr="5" final="true">
            <scm:edge term="C48" to="3"/>
            <scm:edge term="C49" to="3"/>
            <scm:edge term="C50" to="3"/>
            <scm:edge term="C52" to="3"/>
            <scm:edge term="C53" to="3"/>
            <scm:edge term="C55" to="3"/>
            <scm:edge term="C56" to="3"/>
            <scm:edge term="C57" to="3"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C59" to="3"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C62" to="3"/>
            <scm:edge term="C63" to="6"/>
            <scm:edge term="C64" to="3"/>
            <scm:edge term="C65" to="3"/>
            <scm:edge term="C66" to="4"/>
            <scm:edge term="C67" to="3"/>
            <scm:edge term="C68" to="3"/>
            <scm:edge term="C69" to="3"/>
            <scm:edge term="C70" to="2"/>
            <scm:edge term="C72" to="3"/>
         </scm:state>
         <scm:state nr="6" final="true">
            <scm:edge term="C63" to="6"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C5" base="#anySimpleType" variety="list" itemType="C197"/>
   <scm:complexType id="C198"
                    name="localSimpleType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C199"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C200"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C201"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C202" to="1"/>
            <scm:edge term="C203" to="1"/>
            <scm:edge term="C13" to="2"/>
            <scm:edge term="C204" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2">
            <scm:edge term="C202" to="1"/>
            <scm:edge term="C203" to="1"/>
            <scm:edge term="C204" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C187"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="#unnamed"/>
   </scm:simpleType>
   <scm:complexType id="C191"
                    name="facet"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="true" inheritable="false" ref="C205"/>
      <scm:attributeUse required="false" inheritable="false" ref="C193" default="false"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C206" base="C207" variety="atomic" primitiveType="#string">
      <scm:enumeration value="extension"/>
      <scm:enumeration value="substitution"/>
      <scm:enumeration value="restriction"/>
   </scm:simpleType>
   <scm:simpleType id="C208"
                   name="typeDerivationControl"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="C207"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="extension"/>
      <scm:enumeration value="union"/>
      <scm:enumeration value="list"/>
      <scm:enumeration value="restriction"/>
   </scm:simpleType>
   <scm:simpleType id="C209" base="C184" variety="atomic" primitiveType="#string">
      <scm:enumeration value="private"/>
      <scm:enumeration value="public"/>
      <scm:enumeration value="final"/>
   </scm:simpleType>
   <scm:simpleType id="C210" base="#anySimpleType" variety="list" itemType="C211"/>
   <scm:complexType id="C81"
                    name="attributeGroup"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="true"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C175"/>
      <scm:attributeUse required="false" inheritable="false" ref="C176"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C46"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C13" to="3"/>
            <scm:edge term="C70" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C70" to="2"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C70" to="2"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C212" base="#anySimpleType" variety="list" itemType="C208"/>
   <scm:simpleType id="C213"
                   name="qnameListA"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#anySimpleType"
                   variety="list"
                   itemType="C214"/>
   <scm:simpleType id="C215"
                   name="qnameList"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#anySimpleType"
                   variety="list"
                   itemType="C216"/>
   <scm:complexType id="C217"
                    name="complexRestrictionType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C41"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C42"/>
      <scm:attributeWildcard ref="C218"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
               <scm:choice>
                  <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
                     <scm:sequence>
                        <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C43"/>
                        <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C44"/>
                     </scm:sequence>
                  </scm:modelGroupParticle>
               </scm:choice>
            </scm:modelGroupParticle>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C46"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C47"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C13" to="5"/>
            <scm:edge term="C43" to="4"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C63" to="6"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C63" to="6"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="6"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="6"/>
         </scm:state>
         <scm:state nr="4">
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="5" final="true">
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C43" to="4"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C63" to="6"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="6" final="true">
            <scm:edge term="C63" to="6"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C219"
                    name="topLevelSimpleType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C199"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C220"/>
      <scm:attributeUse required="true" inheritable="false" ref="C221"/>
      <scm:attributeWildcard ref="C222"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C201"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C202" to="1"/>
            <scm:edge term="C203" to="1"/>
            <scm:edge term="C13" to="2"/>
            <scm:edge term="C204" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2">
            <scm:edge term="C202" to="1"/>
            <scm:edge term="C203" to="1"/>
            <scm:edge term="C204" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C156"
                   name="expression"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:pattern value=".+"/>
   </scm:simpleType>
   <scm:complexType id="C163"
                    name="text-element-base-type"
                    targetNamespace="http://www.w3.org/1999/XSL/Transform"
                    base="C74"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="simple"
                    simpleType="C164">
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C167"/>
   </scm:complexType>
   <scm:complexType id="C223"
                    name="attribute"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C224"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C225"
                        default="optional"/>
      <scm:attributeUse required="false" inheritable="false" ref="C226"/>
      <scm:attributeUse required="false" inheritable="false" ref="C227"/>
      <scm:attributeUse required="false" inheritable="false" ref="C228"/>
      <scm:attributeUse required="false" inheritable="false" ref="C229"/>
      <scm:attributeUse required="false" inheritable="false" ref="C230"/>
      <scm:attributeUse required="false" inheritable="false" ref="C175"/>
      <scm:attributeUse required="false" inheritable="false" ref="C176"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C66"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C66" to="1"/>
            <scm:edge term="C13" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2" final="true">
            <scm:edge term="C66" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C231"
                    name="intFacet"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C191"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C232"/>
      <scm:attributeUse required="false" inheritable="false" ref="C193" default="false"/>
      <scm:attributeWildcard ref="C233"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C234"
                   name="level"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="multiple"/>
      <scm:enumeration value="any"/>
      <scm:enumeration value="single"/>
   </scm:simpleType>
   <scm:simpleType id="C235"
                   name="_spaceType"
                   base="#NCName"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="preserve"/>
      <scm:enumeration value="default"/>
   </scm:simpleType>
   <scm:simpleType id="C92"
                   name="prefix-or-default"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="#NCName C236"/>
   <scm:simpleType id="C237"
                   name="specialNamespaceList"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="##any"/>
      <scm:enumeration value="##other"/>
   </scm:simpleType>
   <scm:simpleType id="C181" base="#anySimpleType" variety="list" itemType="C186"/>
   <scm:simpleType id="C238"
                   name="prefix-list-or-all"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C91 C239"/>
   <scm:simpleType id="C240"
                   base="C186"
                   variety="union"
                   memberTypes="#NCName #QName C241">
      <scm:pattern value="\c*:\c*"/>
   </scm:simpleType>
   <scm:simpleType id="C242" base="C243" variety="list" itemType="C244">
      <scm:assert>
         <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                        xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                        xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                        test="count($value) = count(distinct-values($value))"
                        defaultNamespace=""
                        xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      </scm:assert>
   </scm:simpleType>
   <scm:simpleType id="C245"
                   name="formChoice"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#NMTOKEN"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="qualified"/>
      <scm:enumeration value="unqualified"/>
   </scm:simpleType>
   <scm:complexType id="C246"
                    name="keybase"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C247"/>
      <scm:attributeUse required="false" inheritable="false" ref="C248"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C249"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="unbounded" ref="C250"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C249" to="1"/>
            <scm:edge term="C13" to="2"/>
         </scm:state>
         <scm:state nr="1">
            <scm:edge term="C250" to="3"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C249" to="1"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C250" to="4"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C250" to="4"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C199"
                    name="simpleType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="true"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C220"/>
      <scm:attributeUse required="false" inheritable="false" ref="C251"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C201"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C202" to="1"/>
            <scm:edge term="C203" to="1"/>
            <scm:edge term="C13" to="2"/>
            <scm:edge term="C204" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2">
            <scm:edge term="C202" to="1"/>
            <scm:edge term="C203" to="1"/>
            <scm:edge term="C204" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C252"
                   name="method"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C253 C240"/>
   <scm:simpleType id="C254"
                   name="EQNames"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="list"
                   itemType="C186"/>
   <scm:simpleType id="C186"
                   name="EQName"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="#NCName #QName C241"/>
   <scm:simpleType id="C255" base="C256" variety="list" itemType="C157">
      <scm:minLength value="1"/>
   </scm:simpleType>
   <scm:simpleType id="C257"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="start"/>
      <scm:enumeration value="end"/>
   </scm:simpleType>
   <scm:simpleType id="C258"
                   base="#NMTOKEN"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="unbounded"/>
   </scm:simpleType>
   <scm:simpleType id="C85"
                   name="char"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#string"
                   variety="atomic"
                   primitiveType="#string">
      <scm:length value="1"/>
   </scm:simpleType>
   <scm:simpleType id="C259"
                   name="yes-or-no"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="no"/>
      <scm:enumeration value="yes"/>
      <scm:enumeration value="false"/>
      <scm:enumeration value="0"/>
      <scm:enumeration value="1"/>
      <scm:enumeration value="true"/>
   </scm:simpleType>
   <scm:simpleType id="C260"
                   name="yes-or-no-or-maybe"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="no"/>
      <scm:enumeration value="maybe"/>
      <scm:enumeration value="yes"/>
      <scm:enumeration value="false"/>
      <scm:enumeration value="0"/>
      <scm:enumeration value="1"/>
      <scm:enumeration value="true"/>
   </scm:simpleType>
   <scm:simpleType id="C261"
                   base="#NMTOKEN"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="suffix"/>
      <scm:enumeration value="interleave"/>
   </scm:simpleType>
   <scm:simpleType id="C262"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="#all"/>
   </scm:simpleType>
   <scm:simpleType id="C214"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="#QName C263"/>
   <scm:simpleType id="C263"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="##defined"/>
   </scm:simpleType>
   <scm:simpleType id="C264"
                   base="#NMTOKEN"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="lax"/>
      <scm:enumeration value="skip"/>
      <scm:enumeration value="strict"/>
   </scm:simpleType>
   <scm:simpleType id="C182"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="#all"/>
   </scm:simpleType>
   <scm:simpleType id="C239"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="#all"/>
   </scm:simpleType>
   <scm:complexType id="C15"
                    name="explicitGroup"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C172"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C173" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C174" default="1"/>
      <scm:attributeWildcard ref="C265"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C266"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C51" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C13" to="2"/>
            <scm:edge term="C61" to="1"/>
            <scm:edge term="C54" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C51" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C61" to="1"/>
            <scm:edge term="C54" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C51" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C61" to="1"/>
            <scm:edge term="C54" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C267"
                   base="#anySimpleType"
                   variety="list"
                   itemType="#QName"/>
   <scm:simpleType id="C268"
                   name="blockSet"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C269 C270"/>
   <scm:simpleType id="C271"
                   name="public"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string"/>
   <scm:complexType id="C272"
                    name="topLevelAttribute"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C223"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C273"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C226"/>
      <scm:attributeUse required="false" inheritable="false" ref="C224"/>
      <scm:attributeUse required="true" inheritable="false" ref="C274"/>
      <scm:attributeUse required="false" inheritable="false" ref="C227"/>
      <scm:attributeWildcard ref="C275"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C66"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C66" to="1"/>
            <scm:edge term="C13" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2" final="true">
            <scm:edge term="C66" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C276"
                   base="C277"
                   variety="union"
                   memberTypes="#nonNegativeInteger C258">
      <scm:enumeration value="0"/>
      <scm:enumeration value="1"/>
   </scm:simpleType>
   <scm:complexType id="C278"
                    name="groupRef"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C171"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C173" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C174" default="1"/>
      <scm:attributeUse required="true" inheritable="false" ref="C279"/>
      <scm:attributeWildcard ref="C280"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C281"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="optional"/>
      <scm:enumeration value="required"/>
      <scm:enumeration value="absent"/>
   </scm:simpleType>
   <scm:simpleType id="C4" base="#token" variety="atomic" primitiveType="#string">
      <scm:enumeration value="#all"/>
   </scm:simpleType>
   <scm:complexType id="C282"
                    name="namedGroup"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C171"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C283"/>
      <scm:attributeWildcard ref="C284"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C285"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C286"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C287"/>
               </scm:choice>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C287" to="1"/>
            <scm:edge term="C285" to="1"/>
            <scm:edge term="C13" to="2"/>
            <scm:edge term="C286" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2">
            <scm:edge term="C287" to="1"/>
            <scm:edge term="C285" to="1"/>
            <scm:edge term="C286" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C288"
                   name="xpathDefaultNamespace"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="#anyURI C183"/>
   <scm:complexType id="C8"
                    name="annotated"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C289"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C290"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string"/>
   <scm:complexType id="C291"
                    name="simpleExtensionType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C178"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C179"/>
      <scm:attributeWildcard ref="C292"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C46"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C47"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C13" to="3"/>
            <scm:edge term="C70" to="2"/>
            <scm:edge term="C63" to="4"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C63" to="4"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C70" to="2"/>
            <scm:edge term="C63" to="4"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="2"/>
            <scm:edge term="C70" to="2"/>
            <scm:edge term="C63" to="4"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C63" to="4"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C149"
                   name="EQName-in-namespace"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="C186"
                   variety="union"
                   memberTypes="#NCName #QName C241">
      <scm:pattern value="Q\{.+\}.+|\i\c*:.+"/>
   </scm:simpleType>
   <scm:complexType id="C172"
                    name="group"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="true"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C175"/>
      <scm:attributeUse required="false" inheritable="false" ref="C176"/>
      <scm:attributeUse required="false" inheritable="false" ref="C174" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C173" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C293"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C54" to="1"/>
            <scm:edge term="C71" to="1"/>
            <scm:edge term="C51" to="1"/>
            <scm:edge term="C61" to="1"/>
            <scm:edge term="C13" to="2"/>
            <scm:edge term="C19" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C54" to="1"/>
            <scm:edge term="C71" to="1"/>
            <scm:edge term="C51" to="1"/>
            <scm:edge term="C61" to="1"/>
            <scm:edge term="C19" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C54" to="1"/>
            <scm:edge term="C71" to="1"/>
            <scm:edge term="C51" to="1"/>
            <scm:edge term="C61" to="1"/>
            <scm:edge term="C19" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C294"
                    name="wildcard"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C295"/>
      <scm:attributeUse required="false" inheritable="false" ref="C296"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C297"
                        default="strict"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C150"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="unclassified"/>
      <scm:enumeration value="shallow-descent"/>
      <scm:enumeration value="deep-descent"/>
      <scm:enumeration value="filter"/>
      <scm:enumeration value="ascent"/>
      <scm:enumeration value="absorbing"/>
      <scm:enumeration value="inspection"/>
   </scm:simpleType>
   <scm:simpleType id="C197" base="C207" variety="atomic" primitiveType="#string">
      <scm:enumeration value="extension"/>
      <scm:enumeration value="union"/>
      <scm:enumeration value="list"/>
      <scm:enumeration value="restriction"/>
   </scm:simpleType>
   <scm:simpleType id="C243" base="#anySimpleType" variety="list" itemType="C244"/>
   <scm:simpleType id="C244"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C186 C146"/>
   <scm:simpleType id="C298"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="#current"/>
      <scm:enumeration value="#default"/>
      <scm:enumeration value="#unnamed"/>
   </scm:simpleType>
   <scm:simpleType id="C277"
                   name="allNNI"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="#nonNegativeInteger C258"/>
   <scm:simpleType id="C1"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C186 C299 C300"/>
   <scm:simpleType id="C299"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="*"/>
   </scm:simpleType>
   <scm:simpleType id="C300"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:pattern value="(\*:\i\c*)|(\i\c*:\*)"/>
   </scm:simpleType>
   <scm:simpleType id="C236"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="#default"/>
   </scm:simpleType>
   <scm:simpleType id="C301"
                   name="modes"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C242 C147"/>
   <scm:simpleType id="C302"
                   name="mode"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C186 C298"/>
   <scm:simpleType id="C303"
                   base="#nonNegativeInteger"
                   variety="atomic"
                   primitiveType="#integer">
      <scm:enumeration value="0"/>
      <scm:enumeration value="1"/>
   </scm:simpleType>
   <scm:simpleType id="C304"
                   base="#NMTOKEN"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="optional"/>
      <scm:enumeration value="required"/>
      <scm:enumeration value="prohibited"/>
   </scm:simpleType>
   <scm:simpleType id="C164" base="C305" variety="atomic" primitiveType="#string"/>
   <scm:simpleType id="C305"
                   base="#string"
                   variety="atomic"
                   primitiveType="#string"/>
   <scm:simpleType id="C256"
                   name="basicNamespaceList"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#anySimpleType"
                   variety="list"
                   itemType="C157"/>
   <scm:simpleType id="C207"
                   name="derivationControl"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#NMTOKEN"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="extension"/>
      <scm:enumeration value="substitution"/>
      <scm:enumeration value="union"/>
      <scm:enumeration value="list"/>
      <scm:enumeration value="restriction"/>
   </scm:simpleType>
   <scm:simpleType id="C306"
                   base="#NMTOKEN"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="none"/>
      <scm:enumeration value="suffix"/>
      <scm:enumeration value="interleave"/>
   </scm:simpleType>
   <scm:simpleType id="C307"
                   name="component-kind-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="template"/>
      <scm:enumeration value="*"/>
      <scm:enumeration value="mode"/>
      <scm:enumeration value="function"/>
      <scm:enumeration value="variable"/>
      <scm:enumeration value="attribute-set"/>
   </scm:simpleType>
   <scm:simpleType id="C308"
                   name="visibility-not-hidden-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="C184"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="private"/>
      <scm:enumeration value="abstract"/>
      <scm:enumeration value="public"/>
      <scm:enumeration value="final"/>
   </scm:simpleType>
   <scm:complexType id="C309"
                    name="noFixedFacet"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C191"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C205"/>
      <scm:attributeWildcard ref="C310"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C311"
                    name="transform-element-base-type"
                    targetNamespace="http://www.w3.org/1999/XSL/Transform"
                    base="C73"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C312"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C313"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C314"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C211"
                   name="reducedDerivationControl"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="C207"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="extension"/>
      <scm:enumeration value="restriction"/>
   </scm:simpleType>
   <scm:complexType id="C74"
                    name="versioned-element-type"
                    targetNamespace="http://www.w3.org/1999/XSL/Transform"
                    base="C23"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C315"
                   name="namespaceList"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C237 C256"/>
   <scm:complexType id="C316"
                    name="topLevelElement"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C317"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C318"/>
      <scm:attributeUse required="false" inheritable="false" ref="C319"/>
      <scm:attributeUse required="false" inheritable="false" ref="C320"/>
      <scm:attributeUse required="false" inheritable="false" ref="C321"/>
      <scm:attributeUse required="false" inheritable="false" ref="C322" default="false"/>
      <scm:attributeUse required="false" inheritable="false" ref="C323"/>
      <scm:attributeUse required="true" inheritable="false" ref="C324"/>
      <scm:attributeUse required="false" inheritable="false" ref="C325"/>
      <scm:attributeUse required="false" inheritable="false" ref="C326"/>
      <scm:attributeWildcard ref="C327"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C66"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C90"/>
               </scm:choice>
            </scm:modelGroupParticle>
            <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C328"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C329"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C90" to="1"/>
            <scm:edge term="C328" to="2"/>
            <scm:edge term="C330" to="3"/>
            <scm:edge term="C331" to="3"/>
            <scm:edge term="C13" to="4"/>
            <scm:edge term="C66" to="1"/>
            <scm:edge term="C332" to="3"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C328" to="2"/>
            <scm:edge term="C330" to="3"/>
            <scm:edge term="C331" to="3"/>
            <scm:edge term="C332" to="3"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C328" to="2"/>
            <scm:edge term="C330" to="3"/>
            <scm:edge term="C331" to="3"/>
            <scm:edge term="C332" to="3"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C330" to="3"/>
            <scm:edge term="C331" to="3"/>
            <scm:edge term="C332" to="3"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C90" to="1"/>
            <scm:edge term="C328" to="2"/>
            <scm:edge term="C330" to="3"/>
            <scm:edge term="C331" to="3"/>
            <scm:edge term="C66" to="1"/>
            <scm:edge term="C332" to="3"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C333"
                    name="simpleExplicitGroup"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C15"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C334"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C266"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C51" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C13" to="2"/>
            <scm:edge term="C61" to="1"/>
            <scm:edge term="C54" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C51" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C61" to="1"/>
            <scm:edge term="C54" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C51" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C61" to="1"/>
            <scm:edge term="C54" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C335"
                   name="item-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:pattern value=".+"/>
   </scm:simpleType>
   <scm:simpleType id="C336"
                   name="uri-list"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#anySimpleType"
                   variety="list"
                   itemType="#anyURI"/>
   <scm:simpleType id="C216"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="#QName C337"/>
   <scm:simpleType id="C337"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="##definedSibling"/>
      <scm:enumeration value="##defined"/>
   </scm:simpleType>
   <scm:complexType id="C338"
                    name="localElement"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C317"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C318"/>
      <scm:attributeUse required="false" inheritable="false" ref="C173" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C320"/>
      <scm:attributeUse required="false" inheritable="false" ref="C339"/>
      <scm:attributeUse required="false" inheritable="false" ref="C340"/>
      <scm:attributeUse required="false" inheritable="false" ref="C174" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C323"/>
      <scm:attributeUse required="false" inheritable="false" ref="C175"/>
      <scm:attributeUse required="false" inheritable="false" ref="C325"/>
      <scm:attributeUse required="false" inheritable="false" ref="C326"/>
      <scm:attributeUse required="false" inheritable="false" ref="C176"/>
      <scm:attributeWildcard ref="C341"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C66"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C90"/>
               </scm:choice>
            </scm:modelGroupParticle>
            <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C328"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C329"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C330" to="1"/>
            <scm:edge term="C331" to="1"/>
            <scm:edge term="C328" to="2"/>
            <scm:edge term="C90" to="3"/>
            <scm:edge term="C332" to="1"/>
            <scm:edge term="C66" to="3"/>
            <scm:edge term="C13" to="4"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C330" to="1"/>
            <scm:edge term="C331" to="1"/>
            <scm:edge term="C332" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C330" to="1"/>
            <scm:edge term="C331" to="1"/>
            <scm:edge term="C328" to="2"/>
            <scm:edge term="C332" to="1"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C330" to="1"/>
            <scm:edge term="C331" to="1"/>
            <scm:edge term="C328" to="2"/>
            <scm:edge term="C332" to="1"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C330" to="1"/>
            <scm:edge term="C331" to="1"/>
            <scm:edge term="C328" to="2"/>
            <scm:edge term="C90" to="3"/>
            <scm:edge term="C332" to="1"/>
            <scm:edge term="C66" to="3"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C317"
                    name="element"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="true"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C320"/>
      <scm:attributeUse required="false" inheritable="false" ref="C321"/>
      <scm:attributeUse required="false" inheritable="false" ref="C318"/>
      <scm:attributeUse required="false" inheritable="false" ref="C325"/>
      <scm:attributeUse required="false" inheritable="false" ref="C323"/>
      <scm:attributeUse required="false" inheritable="false" ref="C322" default="false"/>
      <scm:attributeUse required="false" inheritable="false" ref="C319"/>
      <scm:attributeUse required="false" inheritable="false" ref="C326"/>
      <scm:attributeUse required="false" inheritable="false" ref="C339"/>
      <scm:attributeUse required="false" inheritable="false" ref="C340"/>
      <scm:attributeUse required="false" inheritable="false" ref="C175"/>
      <scm:attributeUse required="false" inheritable="false" ref="C176"/>
      <scm:attributeUse required="false" inheritable="false" ref="C174" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C173" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
                     <scm:choice>
                        <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C66"/>
                        <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C90"/>
                     </scm:choice>
                  </scm:modelGroupParticle>
                  <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C328"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C329"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C90" to="1"/>
            <scm:edge term="C330" to="2"/>
            <scm:edge term="C66" to="1"/>
            <scm:edge term="C331" to="2"/>
            <scm:edge term="C328" to="3"/>
            <scm:edge term="C332" to="2"/>
            <scm:edge term="C13" to="4"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C330" to="2"/>
            <scm:edge term="C331" to="2"/>
            <scm:edge term="C328" to="3"/>
            <scm:edge term="C332" to="2"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C330" to="2"/>
            <scm:edge term="C331" to="2"/>
            <scm:edge term="C332" to="2"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C330" to="2"/>
            <scm:edge term="C331" to="2"/>
            <scm:edge term="C328" to="3"/>
            <scm:edge term="C332" to="2"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C90" to="1"/>
            <scm:edge term="C330" to="2"/>
            <scm:edge term="C66" to="1"/>
            <scm:edge term="C331" to="2"/>
            <scm:edge term="C328" to="3"/>
            <scm:edge term="C332" to="2"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C342"
                   name="avt"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#string"
                   variety="atomic"
                   primitiveType="#string"/>
   <scm:simpleType id="C343"
                   base="#NMTOKEN"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="optional"/>
      <scm:enumeration value="required"/>
      <scm:enumeration value="prohibited"/>
   </scm:simpleType>
   <scm:complexType id="C96"
                    name="sequence-constructor-and-select"
                    targetNamespace="http://www.w3.org/1999/XSL/Transform"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C345"
                   base="#anySimpleType"
                   variety="list"
                   itemType="#QName"/>
   <scm:complexType id="C344"
                    name="sequence-constructor"
                    targetNamespace="http://www.w3.org/1999/XSL/Transform"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C346"
                   name="derivationSet"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C262 C210"/>
   <scm:simpleType id="C269"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="#all"/>
   </scm:simpleType>
   <scm:simpleType id="C270" base="#anySimpleType" variety="list" itemType="C206"/>
   <scm:complexType id="C347"
                    name="complexType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C8"
                    derivationMethod="extension"
                    abstract="true"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C348"/>
      <scm:attributeUse required="false" inheritable="false" ref="C349"/>
      <scm:attributeUse required="false" inheritable="false" ref="C350" default="false"/>
      <scm:attributeUse required="false" inheritable="false" ref="C351"/>
      <scm:attributeUse required="false" inheritable="false" ref="C352"/>
      <scm:attributeUse required="false" inheritable="false" ref="C353" default="true"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C354"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C355" to="4"/>
            <scm:edge term="C13" to="6"/>
            <scm:edge term="C43" to="5"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C63" to="7"/>
            <scm:edge term="C356" to="4"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="4" final="true"/>
         <scm:state nr="5" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C71" to="2"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="6" final="true">
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C355" to="4"/>
            <scm:edge term="C43" to="5"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C63" to="7"/>
            <scm:edge term="C356" to="4"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="7" final="true">
            <scm:edge term="C63" to="7"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C357"
                   name="typed-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="no"/>
      <scm:enumeration value="lax"/>
      <scm:enumeration value="unspecified"/>
      <scm:enumeration value="yes"/>
      <scm:enumeration value="false"/>
      <scm:enumeration value="0"/>
      <scm:enumeration value="1"/>
      <scm:enumeration value="true"/>
      <scm:enumeration value="strict"/>
   </scm:simpleType>
   <scm:simpleType id="C94"
                   name="validation-type"
                   targetNamespace="http://www.w3.org/1999/XSL/Transform"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="lax"/>
      <scm:enumeration value="preserve"/>
      <scm:enumeration value="strip"/>
      <scm:enumeration value="strict"/>
   </scm:simpleType>
   <scm:simpleType id="C158"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="##targetNamespace"/>
      <scm:enumeration value="##local"/>
   </scm:simpleType>
   <scm:complexType id="C358"
                    name="localComplexType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C347"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C349"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C353" default="true"/>
      <scm:attributeWildcard ref="C359"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C354"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C355" to="4"/>
            <scm:edge term="C13" to="5"/>
            <scm:edge term="C43" to="6"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C63" to="7"/>
            <scm:edge term="C356" to="4"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="4" final="true"/>
         <scm:state nr="5" final="true">
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C355" to="4"/>
            <scm:edge term="C43" to="6"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C63" to="7"/>
            <scm:edge term="C356" to="4"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="6" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C71" to="2"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="7" final="true">
            <scm:edge term="C63" to="7"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C360"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="optional"/>
      <scm:enumeration value="required"/>
      <scm:enumeration value="absent"/>
   </scm:simpleType>
   <scm:complexType id="C361"
                    name="topLevelComplexType"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="C347"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C349"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C351"/>
      <scm:attributeUse required="false" inheritable="false" ref="C353" default="true"/>
      <scm:attributeUse required="false" inheritable="false" ref="C350" default="false"/>
      <scm:attributeUse required="true" inheritable="false" ref="C362"/>
      <scm:attributeUse required="false" inheritable="false" ref="C352"/>
      <scm:attributeWildcard ref="C363"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C354"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C355" to="5"/>
            <scm:edge term="C13" to="4"/>
            <scm:edge term="C43" to="6"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C63" to="7"/>
            <scm:edge term="C356" to="5"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C355" to="5"/>
            <scm:edge term="C43" to="6"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C63" to="7"/>
            <scm:edge term="C356" to="5"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C71" to="2"/>
         </scm:state>
         <scm:state nr="5" final="true"/>
         <scm:state nr="6" final="true">
            <scm:edge term="C58" to="1"/>
            <scm:edge term="C54" to="2"/>
            <scm:edge term="C60" to="3"/>
            <scm:edge term="C51" to="2"/>
            <scm:edge term="C61" to="2"/>
            <scm:edge term="C71" to="2"/>
            <scm:edge term="C70" to="3"/>
            <scm:edge term="C63" to="7"/>
         </scm:state>
         <scm:state nr="7" final="true">
            <scm:edge term="C63" to="7"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:simpleType id="C253"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:enumeration value="xhtml"/>
      <scm:enumeration value="xml"/>
      <scm:enumeration value="html"/>
      <scm:enumeration value="text"/>
   </scm:simpleType>
   <scm:simpleType id="C241"
                   base="#token"
                   variety="atomic"
                   primitiveType="#string">
      <scm:pattern value="Q\{[^{}]*\}[\i-[:]][\c-[:]]*"/>
   </scm:simpleType>
   <scm:simpleType id="C364"
                   name="fullDerivationSet"
                   targetNamespace="http://www.w3.org/2001/XMLSchema"
                   base="#anySimpleType"
                   variety="union"
                   memberTypes="C160 C212"/>
   <scm:complexType id="C289"
                    name="openAttrs"
                    targetNamespace="http://www.w3.org/2001/XMLSchema"
                    base="#anyType"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="empty">
      <scm:attributeWildcard ref="C12"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:element id="C365"
                name="override"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C366"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C112"
                name="number"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C367"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C61"
                name="sequence"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C15"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C105"
                name="namespace"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C368"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C71"
                name="all"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C14"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C53"
                name="assertion"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C7"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C62"
                name="maxExclusive"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C191"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C52"
                name="maxInclusive"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C191"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C142"
                name="analyze-string"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C369"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C250"
                name="field"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C370"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C203"
                name="restriction"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C371"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C119"
                name="comment"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C95"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C372"
                name="decimal-format"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C373"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C116"
                name="element"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C375"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C376"
                name="preserve-space"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C377"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C68"
                name="whiteSpace"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C378"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C122"
                name="perform-sort"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C379"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C380"
                name="accumulator"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C381"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C382"
                name="param"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C383"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C106"
                name="where-populated"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C344"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C102"
                name="for-each-group"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C384"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C385"
                name="accumulator-rule"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C386"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C55"
                name="minLength"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C190"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C387"
                name="accept"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C388"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C138"
                name="assert"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C389"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C390"
                name="character-map"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C391"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C141"
                name="message"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C392"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C393"
                name="when"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C394"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C374"
                name="declaration"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C23"
                global="true"
                nillable="false"
                abstract="true"/>
   <scm:element id="C43"
                name="openContent"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C395"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C135"
                name="on-non-empty"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C95"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C396"
                name="transform"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C397"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C114"
                name="call-template"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C398"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C107"
                name="fallback"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C344"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C115"
                name="variable"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C399"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C400"
                name="redefine"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C401"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C331"
                name="key"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C246"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C49"
                name="facet"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="#anyType"
                global="true"
                nillable="false"
                abstract="true"/>
   <scm:element id="C402"
                name="merge-source"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C403"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C332"
                name="keyref"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C404"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C101"
                name="next-match"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C405"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C125"
                name="try"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C406"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C50"
                name="minExclusive"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C191"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C65"
                name="minInclusive"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C191"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C407"
                name="merge-action"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C344"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C67"
                name="enumeration"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C309"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C408"
                name="namespace-alias"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C409"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C111"
                name="copy-of"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C410"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C118"
                name="iterate"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C411"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C412"
                name="output"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C413"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C414"
                name="attributeGroup"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C168"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C129"
                name="copy"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C415"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C416"
                name="sort"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C417"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C418"
                name="matching-substring"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C344"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C20"
                name="any"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C419"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C140"
                name="sequence"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C95"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C249"
                name="selector"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C420"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C127"
                name="apply-imports"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C421"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C59"
                name="pattern"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C422"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C108"
                name="apply-templates"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C423"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C424"
                name="key"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C425"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C426"
                name="with-param"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C427"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C139"
                name="if"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C428"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C130"
                name="processing-instruction"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C429"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C430"
                name="attribute"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C272"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C431"
                name="package"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C432"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C356"
                name="complexContent"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C433"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C434"
                name="documentation"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C435"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C436"
                name="context-item"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C437"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C57"
                name="length"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C190"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C438"
                name="simpleType"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C219"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C136"
                name="choose"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C439"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C124"
                name="merge"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C440"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C109"
                name="map-entry"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C441"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C132"
                name="merge-key"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C442"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C443"
                name="override"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C444"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C69"
                name="maxLength"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C190"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C330"
                name="unique"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C246"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C121"
                name="fork"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C445"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C137"
                name="for-each"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C446"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C143"
                name="instruction"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C74"
                global="true"
                nillable="false"
                abstract="true"/>
   <scm:element id="C447"
                name="non-matching-substring"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C344"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C58"
                name="anyAttribute"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C448"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C449"
                name="include"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C450"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C123"
                name="attribute"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C451"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C452"
                name="catch"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C453"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C113"
                name="result-document"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C454"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C13"
                name="annotation"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C455"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C456"
                name="group"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C282"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C204"
                name="union"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C457"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C458"
                name="import"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C459"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C131"
                name="break"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C95"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C460"
                name="on-completion"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C95"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C461"
                name="strip-space"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C462"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C463"
                name="template"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C464"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C465"
                name="import"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C466"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C355"
                name="simpleContent"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C467"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C468"
                name="mode"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C469"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C64"
                name="fractionDigits"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C190"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C56"
                name="explicitTimezone"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C470"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C471"
                name="include"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C472"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C104"
                name="source-document"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C473"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C474"
                name="schema"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C475"
                global="true"
                nillable="false"
                abstract="false">
      <scm:identityConstraint ref="C476"/>
      <scm:identityConstraint ref="C477"/>
      <scm:identityConstraint ref="C478"/>
      <scm:identityConstraint ref="C479"/>
      <scm:identityConstraint ref="C480"/>
      <scm:identityConstraint ref="C481"/>
      <scm:identityConstraint ref="C482"/>
   </scm:element>
   <scm:element id="C483"
                name="function"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C484"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C133"
                name="value-of"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C485"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C486"
                name="global-context-item"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C487"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C120"
                name="next-iteration"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C488"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C489"
                name="notation"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C490"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C491"
                name="attribute-set"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C492"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C493"
                name="expose"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C494"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C495"
                name="defaultOpenContent"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C496"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C126"
                name="evaluate"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C497"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C498"
                name="element"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C316"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C202"
                name="list"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C499"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C51"
                name="choice"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C15"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C500"
                name="appinfo"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C501"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C110"
                name="text"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C162"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C117"
                name="literal-result-element"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="#anyType"
                global="true"
                nillable="false"
                abstract="true"/>
   <scm:element id="C134"
                name="document"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C502"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C503"
                name="output-character"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C504"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C505"
                name="use-package"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C506"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C507"
                name="complexType"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C361"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C508"
                name="import-schema"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C509"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C374"/>
   </scm:element>
   <scm:element id="C510"
                name="otherwise"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C344"
                global="true"
                nillable="false"
                abstract="false"/>
   <scm:element id="C103"
                name="on-empty"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C95"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C48"
                name="totalDigits"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C511"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C49"/>
   </scm:element>
   <scm:element id="C128"
                name="map"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C344"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C143"/>
   </scm:element>
   <scm:element id="C512"
                name="stylesheet"
                targetNamespace="http://www.w3.org/1999/XSL/Transform"
                type="C397"
                global="true"
                nillable="false"
                abstract="false">
      <scm:substitutionGroupAffiliation ref="C396"/>
   </scm:element>
   <scm:attribute id="C513"
                  name="space"
                  targetNamespace="http://www.w3.org/XML/1998/namespace"
                  type="C235"
                  global="true"
                  inheritable="false"/>
   <scm:attribute id="C514"
                  name="lang"
                  targetNamespace="http://www.w3.org/XML/1998/namespace"
                  type="C188"
                  global="true"
                  inheritable="false"/>
   <scm:attribute id="C515"
                  name="base"
                  targetNamespace="http://www.w3.org/XML/1998/namespace"
                  type="#anyURI"
                  global="true"
                  inheritable="false"/>
   <scm:attribute id="C516"
                  name="id"
                  targetNamespace="http://www.w3.org/XML/1998/namespace"
                  type="#ID"
                  global="true"
                  inheritable="false"/>
   <scm:modelGroupDefinition id="C517"
                             name="redefinable"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C438"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C507"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C456"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C414"/>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C47"
                             name="assertions"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C63"/>
         </scm:sequence>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C518"
                             name="schemaTop"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C517"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C498"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C430"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C489"/>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C293"
                             name="particle"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C19"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C54"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C71"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C51"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C61"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C20"/>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C201"
                             name="simpleDerivation"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C203"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C202"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C204"/>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C100"
                             name="sequence-constructor-group"
                             targetNamespace="http://www.w3.org/1999/XSL/Transform">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C143"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C519"/>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C46"
                             name="attrDecls"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C70"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C60"/>
               </scm:choice>
            </scm:modelGroupParticle>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C58"/>
         </scm:sequence>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C354"
                             name="complexTypeModel"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C355"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C356"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C43"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="1" ref="C44"/>
                  <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C46"/>
                  <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C47"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C266"
                             name="nestedParticle"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C19"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C54"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C51"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C61"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C20"/>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C519"
                             name="result-elements"
                             targetNamespace="http://www.w3.org/1999/XSL/Transform">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C117"/>
            <scm:elementWildcard minOccurs="1" maxOccurs="1" ref="C145"/>
            <scm:elementWildcard minOccurs="1" maxOccurs="1" ref="C144"/>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C45"
                             name="simpleRestrictionModel"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C66"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C49"/>
                  <scm:elementWildcard minOccurs="1" maxOccurs="1" ref="C72"/>
               </scm:choice>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C520"
                             name="allModel"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C19"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C20"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C21"/>
               </scm:choice>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C329"
                             name="identityConstraint"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C330"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C331"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C332"/>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C521"
                             name="composition"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C471"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C458"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C400"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C365"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C13"/>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:modelGroupDefinition id="C44"
                             name="typeDefParticle"
                             targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C54"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C71"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C51"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C61"/>
         </scm:choice>
      </scm:modelGroupParticle>
   </scm:modelGroupDefinition>
   <scm:attributeGroup id="C522"
                       name="occurs"
                       targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:attributeUse required="false" inheritable="false" ref="C174" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C173" default="1"/>
   </scm:attributeGroup>
   <scm:attributeGroup id="C523"
                       name="anyAttrGroup"
                       targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:attributeUse required="false" inheritable="false" ref="C295"/>
      <scm:attributeUse required="false" inheritable="false" ref="C296"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C297"
                        default="strict"/>
   </scm:attributeGroup>
   <scm:attributeGroup id="C524"
                       name="literal-result-element-attributes"
                       targetNamespace="http://www.w3.org/1999/XSL/Transform">
      <scm:attributeUse required="false" inheritable="false" ref="C525"/>
      <scm:attributeUse required="false" inheritable="false" ref="C526"/>
      <scm:attributeUse required="false" inheritable="false" ref="C527" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C528"/>
      <scm:attributeUse required="false" inheritable="false" ref="C529"/>
      <scm:attributeUse required="false" inheritable="false" ref="C530"/>
      <scm:attributeUse required="false" inheritable="false" ref="C531"/>
      <scm:attributeUse required="false" inheritable="false" ref="C532" default="yes"/>
      <scm:attributeUse required="false" inheritable="false" ref="C533" default=""/>
      <scm:attributeUse required="false" inheritable="false" ref="C534"/>
      <scm:attributeUse required="false" inheritable="false" ref="C535"/>
      <scm:attributeUse required="false" inheritable="false" ref="C536"/>
      <scm:attributeUse required="false" inheritable="false" ref="C537"/>
   </scm:attributeGroup>
   <scm:attributeGroup id="C538"
                       name="defRef"
                       targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:attributeUse required="false" inheritable="false" ref="C175"/>
      <scm:attributeUse required="false" inheritable="false" ref="C176"/>
   </scm:attributeGroup>
   <scm:notation id="C539"
                 name="XMLSchemaStructures"
                 targetNamespace="http://www.w3.org/2001/XMLSchema"
                 publicId="structures"
                 systemId="http://www.w3.org/2000/08/XMLSchema.xsd"/>
   <scm:notation id="C540"
                 name="XML"
                 targetNamespace="http://www.w3.org/2001/XMLSchema"
                 publicId="REC-xml-19980210"
                 systemId="http://www.w3.org/TR/1998/REC-xml-19980210"/>
   <scm:wildcard id="C40"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/1999/XSL/Transform"/>
   <scm:attribute id="C35"
                  name="_exclude-result-prefixes"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:attribute id="C28"
                  name="expand-text"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:attribute id="C229"
                  name="targetNamespace"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C223"/>
   <scm:attribute id="C323"
                  name="nillable"
                  type="#boolean"
                  global="false"
                  inheritable="false"
                  containingComplexType="C317"/>
   <scm:attribute id="C352"
                  name="block"
                  type="C346"
                  global="false"
                  inheritable="false"
                  containingComplexType="C347"/>
   <scm:complexType id="C367"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C541"/>
      <scm:attributeUse required="false" inheritable="false" ref="C542"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C543"
                        default="single"/>
      <scm:attributeUse required="false" inheritable="false" ref="C544"/>
      <scm:attributeUse required="false" inheritable="false" ref="C545"/>
      <scm:attributeUse required="false" inheritable="false" ref="C546" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C547"/>
      <scm:attributeUse required="false" inheritable="false" ref="C548"/>
      <scm:attributeUse required="false" inheritable="false" ref="C549"/>
      <scm:attributeUse required="false" inheritable="false" ref="C550"/>
      <scm:attributeUse required="false" inheritable="false" ref="C551"/>
      <scm:attributeUse required="false" inheritable="false" ref="C552"/>
      <scm:attributeUse required="false" inheritable="false" ref="C553"/>
      <scm:attributeUse required="false" inheritable="false" ref="C554"/>
      <scm:attributeUse required="false" inheritable="false" ref="C555"/>
      <scm:attributeUse required="false" inheritable="false" ref="C556"/>
      <scm:attributeUse required="false" inheritable="false" ref="C557"/>
      <scm:attributeUse required="false" inheritable="false" ref="C558"/>
      <scm:attributeUse required="false" inheritable="false" ref="C559"/>
      <scm:attributeUse required="false" inheritable="false" ref="C560"/>
      <scm:attributeUse required="false" inheritable="false" ref="C561"/>
      <scm:attributeUse required="false" inheritable="false" ref="C562"/>
      <scm:attributeUse required="false" inheritable="false" ref="C563"/>
      <scm:attributeUse required="false" inheritable="false" ref="C564"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="if (exists(@value | @_value)) then empty((@select | @_select, @count | @_count, @from | @_from))              and (exists(@_level) or normalize-space(@level)='single') else true()"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C541"
                  name="value"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C557"
                  name="_from"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:complexType id="C467"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C565"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C566"/>
               </scm:choice>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C565" to="1"/>
            <scm:edge term="C566" to="1"/>
            <scm:edge term="C13" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2">
            <scm:edge term="C565" to="1"/>
            <scm:edge term="C566" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C545"
                  name="from"
                  type="C155"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C98"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C96"/>
   <scm:complexType id="C383"
                    base="C95"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C567"/>
      <scm:attributeUse required="false" inheritable="false" ref="C568"/>
      <scm:attributeUse required="false" inheritable="false" ref="C569"/>
      <scm:attributeUse required="false" inheritable="false" ref="C570"/>
      <scm:attributeUse required="false" inheritable="false" ref="C571"/>
      <scm:attributeUse required="false" inheritable="false" ref="C572"/>
      <scm:attributeUse required="false" inheritable="false" ref="C573"/>
      <scm:attributeUse required="false" inheritable="false" ref="C574"/>
      <scm:attributeUse required="false" inheritable="false" ref="C575"/>
      <scm:attributeUse required="false" inheritable="false" ref="C576"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="if (normalize-space(@static) = ('yes', 'true', '1'))                            then empty((*,text()))                            else true()"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:complexType id="C391"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C577"/>
      <scm:attributeUse required="false" inheritable="false" ref="C578" default=""/>
      <scm:attributeUse required="false" inheritable="false" ref="C579"/>
      <scm:attributeUse required="false" inheritable="false" ref="C580"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C503"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C503" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C503" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C193"
                  name="fixed"
                  type="#boolean"
                  global="false"
                  inheritable="false"
                  containingComplexType="C191"/>
   <scm:attribute id="C30"
                  name="use-when"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:attribute id="C325"
                  name="fixed"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C317"/>
   <scm:complexType id="C499"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C581"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C66"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
            <scm:edge term="C66" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C66" to="2"/>
         </scm:state>
         <scm:state nr="2" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C410"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C582"/>
      <scm:attributeUse required="false" inheritable="false" ref="C583" default="no"/>
      <scm:attributeUse required="false" inheritable="false" ref="C584" default="yes"/>
      <scm:attributeUse required="false" inheritable="false" ref="C585"/>
      <scm:attributeUse required="false" inheritable="false" ref="C586"/>
      <scm:attributeUse required="false" inheritable="false" ref="C587"/>
      <scm:attributeUse required="false" inheritable="false" ref="C588"/>
      <scm:attributeUse required="false" inheritable="false" ref="C589"/>
      <scm:attributeUse required="false" inheritable="false" ref="C590"/>
      <scm:attributeUse required="false" inheritable="false" ref="C591"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@type | @_type) and exists(@validation | @_validation))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@select | @_select)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C567"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C383"/>
   <scm:attribute id="C312"
                  name="_version"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C311"/>
   <scm:attribute id="C192"
                  name="value"
                  type="#nonNegativeInteger"
                  global="false"
                  inheritable="false"
                  containingComplexType="C190"/>
   <scm:attribute id="C17"
                  name="minOccurs"
                  type="C303"
                  global="false"
                  inheritable="false"
                  containingComplexType="C14"/>
   <scm:attribute id="C526"
                  name="default-mode"
                  type="C185"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C544"
                  name="count"
                  type="C155"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C535"
                  name="version"
                  targetNamespace="http://www.w3.org/1999/XSL/Transform"
                  type="#decimal"
                  global="false"
                  inheritable="false"/>
   <scm:complexType id="C403"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C592"/>
      <scm:attributeUse required="false" inheritable="false" ref="C593"/>
      <scm:attributeUse required="false" inheritable="false" ref="C594"/>
      <scm:attributeUse required="false" inheritable="false" ref="C595"/>
      <scm:attributeUse required="false" inheritable="false" ref="C596"/>
      <scm:attributeUse required="false" inheritable="false" ref="C597"/>
      <scm:attributeUse required="false" inheritable="false" ref="C598"/>
      <scm:attributeUse required="false" inheritable="false" ref="C599"/>
      <scm:attributeUse required="false" inheritable="false" ref="C600"/>
      <scm:attributeUse required="false" inheritable="false" ref="C601"/>
      <scm:attributeUse required="false" inheritable="false" ref="C602"/>
      <scm:attributeUse required="false" inheritable="false" ref="C603"/>
      <scm:attributeUse required="false" inheritable="false" ref="C604"/>
      <scm:attributeUse required="false" inheritable="false" ref="C605"/>
      <scm:attributeUse required="false" inheritable="false" ref="C606"/>
      <scm:attributeUse required="false" inheritable="false" ref="C607"/>
      <scm:attributeUse required="false" inheritable="false" ref="C608"/>
      <scm:attributeUse required="false" inheritable="false" ref="C609"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:elementParticle minOccurs="1" maxOccurs="unbounded" ref="C132"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C132" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C132" to="2"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C132" to="2"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@select | @_select)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C596"
                  name="streamable"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:complexType id="C423"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C610"
                        default="child::node()"/>
      <scm:attributeUse required="false" inheritable="false" ref="C611"/>
      <scm:attributeUse required="false" inheritable="false" ref="C612"/>
      <scm:attributeUse required="false" inheritable="false" ref="C613"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C416"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C426"/>
         </scm:choice>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C416" to="1"/>
            <scm:edge term="C426" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C416" to="1"/>
            <scm:edge term="C426" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $e in subsequence(xsl:sort, 2) satisfies empty($e/(@stable | @_stable))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C610"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C423"/>
   <scm:complexType id="C475"
                    base="C289"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C614"/>
      <scm:attributeUse required="false" inheritable="false" ref="C615"/>
      <scm:attributeUse required="false" inheritable="false" ref="C616" default=""/>
      <scm:attributeUse required="false" inheritable="false" ref="C617" default=""/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C618"
                        default="unqualified"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C619"
                        default="unqualified"/>
      <scm:attributeUse required="false" inheritable="false" ref="C620"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C621"
                        default="##local"/>
      <scm:attributeUse required="false" inheritable="false" ref="C622"/>
      <scm:attributeUse required="false" inheritable="false" ref="C514"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C521"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
                     <scm:sequence>
                        <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C495"/>
                        <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C13"/>
                     </scm:sequence>
                  </scm:modelGroupParticle>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
                     <scm:sequence>
                        <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C518"/>
                        <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C13"/>
                     </scm:sequence>
                  </scm:modelGroupParticle>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C495" to="3"/>
            <scm:edge term="C400" to="1"/>
            <scm:edge term="C13" to="1"/>
            <scm:edge term="C498" to="2"/>
            <scm:edge term="C456" to="2"/>
            <scm:edge term="C507" to="2"/>
            <scm:edge term="C414" to="2"/>
            <scm:edge term="C471" to="1"/>
            <scm:edge term="C489" to="2"/>
            <scm:edge term="C438" to="2"/>
            <scm:edge term="C365" to="1"/>
            <scm:edge term="C430" to="2"/>
            <scm:edge term="C458" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C495" to="3"/>
            <scm:edge term="C400" to="1"/>
            <scm:edge term="C13" to="1"/>
            <scm:edge term="C498" to="2"/>
            <scm:edge term="C456" to="2"/>
            <scm:edge term="C507" to="2"/>
            <scm:edge term="C414" to="2"/>
            <scm:edge term="C471" to="1"/>
            <scm:edge term="C489" to="2"/>
            <scm:edge term="C438" to="2"/>
            <scm:edge term="C365" to="1"/>
            <scm:edge term="C430" to="2"/>
            <scm:edge term="C458" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C489" to="2"/>
            <scm:edge term="C414" to="2"/>
            <scm:edge term="C430" to="2"/>
            <scm:edge term="C507" to="2"/>
            <scm:edge term="C13" to="5"/>
            <scm:edge term="C438" to="2"/>
            <scm:edge term="C498" to="2"/>
            <scm:edge term="C456" to="2"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C489" to="2"/>
            <scm:edge term="C414" to="2"/>
            <scm:edge term="C430" to="2"/>
            <scm:edge term="C507" to="2"/>
            <scm:edge term="C438" to="2"/>
            <scm:edge term="C498" to="2"/>
            <scm:edge term="C13" to="4"/>
            <scm:edge term="C456" to="2"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C489" to="2"/>
            <scm:edge term="C414" to="2"/>
            <scm:edge term="C430" to="2"/>
            <scm:edge term="C507" to="2"/>
            <scm:edge term="C438" to="2"/>
            <scm:edge term="C498" to="2"/>
            <scm:edge term="C13" to="4"/>
            <scm:edge term="C456" to="2"/>
         </scm:state>
         <scm:state nr="5" final="true">
            <scm:edge term="C489" to="2"/>
            <scm:edge term="C414" to="2"/>
            <scm:edge term="C430" to="2"/>
            <scm:edge term="C507" to="2"/>
            <scm:edge term="C13" to="5"/>
            <scm:edge term="C438" to="2"/>
            <scm:edge term="C498" to="2"/>
            <scm:edge term="C456" to="2"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C614"
                  name="targetNamespace"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C475"/>
   <scm:attribute id="C29"
                  name="extension-element-prefixes"
                  type="C91"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:attribute id="C590"
                  name="_type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C410"/>
   <scm:attribute id="C536"
                  name="type"
                  targetNamespace="http://www.w3.org/1999/XSL/Transform"
                  type="C186"
                  global="false"
                  inheritable="false"/>
   <scm:wildcard id="C310"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C547"
                  name="lang"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:complexType id="C417"
                    base="C95"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C623"/>
      <scm:attributeUse required="false" inheritable="false" ref="C624" default="text"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C625"
                        default="ascending"/>
      <scm:attributeUse required="false" inheritable="false" ref="C626"/>
      <scm:attributeUse required="false" inheritable="false" ref="C627"/>
      <scm:attributeUse required="false" inheritable="false" ref="C628"/>
      <scm:attributeUse required="false" inheritable="false" ref="C629"/>
      <scm:attributeUse required="false" inheritable="false" ref="C630"/>
      <scm:attributeUse required="false" inheritable="false" ref="C631"/>
      <scm:attributeUse required="false" inheritable="false" ref="C632"/>
      <scm:attributeUse required="false" inheritable="false" ref="C633"/>
      <scm:attributeUse required="false" inheritable="false" ref="C634"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C632"
                  name="_case-order"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:attribute id="C626"
                  name="case-order"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:attribute id="C295"
                  name="namespace"
                  type="C315"
                  global="false"
                  inheritable="false"/>
   <scm:complexType id="C397"
                    base="C311"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C635"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C636"
                        default="unspecified"/>
      <scm:attributeUse required="false" inheritable="false" ref="C637"/>
      <scm:attributeUse required="false" inheritable="false" ref="C638"/>
      <scm:attributeUse required="false" inheritable="false" ref="C312"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C313"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C314"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C374"/>
            <scm:elementWildcard minOccurs="1" maxOccurs="1" ref="C639"/>
         </scm:choice>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C376" to="1"/>
            <scm:edge term="C491" to="1"/>
            <scm:edge term="C505" to="1"/>
            <scm:edge term="C508" to="1"/>
            <scm:edge term="C390" to="1"/>
            <scm:edge term="C486" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C412" to="1"/>
            <scm:edge term="C374" to="1"/>
            <scm:edge term="C461" to="1"/>
            <scm:edge term="C465" to="1"/>
            <scm:edge term="C468" to="1"/>
            <scm:edge term="C463" to="1"/>
            <scm:edge term="C483" to="1"/>
            <scm:edge term="C380" to="1"/>
            <scm:edge term="C449" to="1"/>
            <scm:edge term="C408" to="1"/>
            <scm:edge term="C382" to="1"/>
            <scm:edge term="C372" to="1"/>
            <scm:edge term="C424" to="1"/>
            <scm:edge term="C639" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C376" to="1"/>
            <scm:edge term="C491" to="1"/>
            <scm:edge term="C505" to="1"/>
            <scm:edge term="C508" to="1"/>
            <scm:edge term="C390" to="1"/>
            <scm:edge term="C486" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C412" to="1"/>
            <scm:edge term="C374" to="1"/>
            <scm:edge term="C461" to="1"/>
            <scm:edge term="C465" to="1"/>
            <scm:edge term="C468" to="1"/>
            <scm:edge term="C463" to="1"/>
            <scm:edge term="C483" to="1"/>
            <scm:edge term="C380" to="1"/>
            <scm:edge term="C449" to="1"/>
            <scm:edge term="C408" to="1"/>
            <scm:edge term="C382" to="1"/>
            <scm:edge term="C372" to="1"/>
            <scm:edge term="C424" to="1"/>
            <scm:edge term="C639" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $v in (.//xsl:param, .//xsl:variable)[exists(@static | @_static)]             satisfies             $v[parent::xsl:stylesheet or parent::xsl:transform or parent::xsl:override]             "
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $prefix in (@exclude-result-prefixes[not(. = '#all')], @extension-element-prefixes)             satisfies ((if ($prefix = '#default') then '' else $prefix) = in-scope-prefixes(.))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:complexType id="C455"
                    base="C289"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C640"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C500"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C434"/>
         </scm:choice>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C500" to="1"/>
            <scm:edge term="C434" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C500" to="1"/>
            <scm:edge term="C434" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C27"
                  name="exclude-result-prefixes"
                  type="C238"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:attribute id="C169"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C168"/>
   <scm:attribute id="C31"
                  name="xpath-default-namespace"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:complexType id="C419"
                    base="C294"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C641"/>
      <scm:attributeUse required="false" inheritable="false" ref="C174" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C173" default="1"/>
      <scm:attributeUse required="false" inheritable="false" ref="C295"/>
      <scm:attributeUse required="false" inheritable="false" ref="C296"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C297"
                        default="strict"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:wildcard id="C153" processContents="lax" constraint="any"/>
   <scm:attribute id="C598"
                  name="sort-before-merge"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C555"
                  name="_level"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C232"
                  name="value"
                  type="#integer"
                  global="false"
                  inheritable="false"
                  containingComplexType="C231"/>
   <scm:key id="C479"
            name="group"
            targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:selector xmlns:xs="http://www.w3.org/2001/XMLSchema"
                    xpath="xs:group"
                    defaultNamespace=""/>
      <scm:field xmlns:xs="http://www.w3.org/2001/XMLSchema"
                 xpath="@name"
                 defaultNamespace=""/>
   </scm:key>
   <scm:attribute id="C33"
                  name="_default-mode"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:attribute id="C552"
                  name="grouping-size"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:complexType id="C404"
                    base="C246"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C642"/>
      <scm:attributeUse required="false" inheritable="false" ref="C247"/>
      <scm:attributeUse required="false" inheritable="false" ref="C248"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="1">
                     <scm:sequence>
                        <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C249"/>
                        <scm:elementParticle minOccurs="1" maxOccurs="unbounded" ref="C250"/>
                     </scm:sequence>
                  </scm:modelGroupParticle>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C249" to="1"/>
            <scm:edge term="C13" to="2"/>
         </scm:state>
         <scm:state nr="1">
            <scm:edge term="C250" to="3"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C249" to="1"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C250" to="4"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C250" to="4"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C318"
                  name="default"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C317"/>
   <scm:key id="C481"
            name="notation"
            targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:selector xmlns:xs="http://www.w3.org/2001/XMLSchema"
                    xpath="xs:notation"
                    defaultNamespace=""/>
      <scm:field xmlns:xs="http://www.w3.org/2001/XMLSchema"
                 xpath="@name"
                 defaultNamespace=""/>
   </scm:key>
   <scm:attribute id="C248"
                  name="ref"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C246"/>
   <scm:attribute id="C624"
                  name="data-type"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:attribute id="C532"
                  name="inherit-namespaces"
                  targetNamespace="http://www.w3.org/1999/XSL/Transform"
                  type="C259"
                  global="false"
                  inheritable="false"/>
   <scm:wildcard id="C218"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C36"
                  name="_expand-text"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:attribute id="C228"
                  name="form"
                  type="C245"
                  global="false"
                  inheritable="false"
                  containingComplexType="C223"/>
   <scm:attribute id="C525"
                  name="default-collation"
                  targetNamespace="http://www.w3.org/1999/XSL/Transform"
                  type="C336"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C606"
                  name="_use-accumulators"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:complexType id="C451"
                    base="C95"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C643"/>
      <scm:attributeUse required="false" inheritable="false" ref="C644"/>
      <scm:attributeUse required="false" inheritable="false" ref="C645"/>
      <scm:attributeUse required="false" inheritable="false" ref="C646"/>
      <scm:attributeUse required="false" inheritable="false" ref="C647"/>
      <scm:attributeUse required="false" inheritable="false" ref="C648"/>
      <scm:attributeUse required="false" inheritable="false" ref="C649"/>
      <scm:attributeUse required="false" inheritable="false" ref="C650"/>
      <scm:attributeUse required="false" inheritable="false" ref="C651"/>
      <scm:attributeUse required="false" inheritable="false" ref="C652"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@type | @_type) and exists(@validation | @_validation))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C560"
                  name="_letter-value"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:key id="C482"
            name="identityConstraint"
            targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:selector xmlns:xs="http://www.w3.org/2001/XMLSchema"
                    xpath=".//xs:key|.//xs:unique|.//xs:keyref"
                    defaultNamespace=""/>
      <scm:field xmlns:xs="http://www.w3.org/2001/XMLSchema"
                 xpath="@name"
                 defaultNamespace=""/>
   </scm:key>
   <scm:attribute id="C633"
                  name="_collation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:attribute id="C87"
                  name="test"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C86"/>
   <scm:complexType id="C504"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C653"/>
      <scm:attributeUse required="false" inheritable="false" ref="C654"/>
      <scm:attributeUse required="false" inheritable="false" ref="C655"/>
      <scm:attributeUse required="false" inheritable="false" ref="C656"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@character | @_character)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@string | @_string)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:complexType id="C502"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C657"/>
      <scm:attributeUse required="false" inheritable="false" ref="C658"/>
      <scm:attributeUse required="false" inheritable="false" ref="C659"/>
      <scm:attributeUse required="false" inheritable="false" ref="C660"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@type | @_type) and exists(@validation | @_validation))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:complexType id="C405"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C426"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C107"/>
         </scm:choice>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C426" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C426" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C611"
                  name="mode"
                  type="C302"
                  global="false"
                  inheritable="false"
                  containingComplexType="C423"/>
   <scm:attribute id="C561"
                  name="_ordinal"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C173"
                  name="maxOccurs"
                  type="C277"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C165"
                  name="disable-output-escaping"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C162"/>
   <scm:wildcard id="C83"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C654"
                  name="string"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C504"/>
   <scm:attribute id="C220"
                  name="final"
                  type="C3"
                  global="false"
                  inheritable="false"
                  containingComplexType="C199"/>
   <scm:complexType id="C459"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C661"/>
      <scm:attributeUse required="false" inheritable="false" ref="C662"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C582"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C410"/>
   <scm:attribute id="C594"
                  name="for-each-source"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C546"
                  name="format"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:wildcard id="C18"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C24"
                  name="default-collation"
                  type="C336"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:complexType id="C506"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C663"/>
      <scm:attributeUse required="false" inheritable="false" ref="C664"/>
      <scm:attributeUse required="false" inheritable="false" ref="C665"/>
      <scm:attributeUse required="false" inheritable="false" ref="C666"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C387"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C443"/>
         </scm:choice>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C387" to="1"/>
            <scm:edge term="C443" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C387" to="1"/>
            <scm:edge term="C443" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C656"
                  name="_string"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C504"/>
   <scm:attribute id="C627"
                  name="collation"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:complexType id="C464"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C667"/>
      <scm:attributeUse required="false" inheritable="false" ref="C668"/>
      <scm:attributeUse required="false" inheritable="false" ref="C669"/>
      <scm:attributeUse required="false" inheritable="false" ref="C670"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C671"
                        default="item()*"/>
      <scm:attributeUse required="false" inheritable="false" ref="C672"/>
      <scm:attributeUse required="false" inheritable="false" ref="C673"/>
      <scm:attributeUse required="false" inheritable="false" ref="C674"/>
      <scm:attributeUse required="false" inheritable="false" ref="C675"/>
      <scm:attributeUse required="false" inheritable="false" ref="C676"/>
      <scm:attributeUse required="false" inheritable="false" ref="C677"/>
      <scm:attributeUse required="false" inheritable="false" ref="C678"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C436"/>
                  <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C382"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C382" to="2"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C436" to="3"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C382" to="2"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C382" to="2"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@visibility | @_visibility) and empty(@name | @_name))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="if (normalize-space(@visibility) = 'abstract') then empty(* except (xsl:context-item, xsl:param)) else true()"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $e in xsl:param satisfies empty($e/(@visibility | @_visibility))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@match | @_match) or exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="if (empty(@match | @_match)) then (empty(@mode | @_mode) and empty(@priority | @_priority)) else true()"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(normalize-space(@visibility) = 'abstract' and exists(@match))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C675"
                  name="_mode"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C668"
                  name="priority"
                  type="#decimal"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C672"
                  name="visibility"
                  type="C184"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C674"
                  name="_priority"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C677"
                  name="_as"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C667"
                  name="match"
                  type="C155"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C587"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C410"/>
   <scm:attribute id="C82"
                  name="ref"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C80"/>
   <scm:attribute id="C669"
                  name="mode"
                  type="C301"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:wildcard id="C359"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:complexType id="C469"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C679"/>
      <scm:attributeUse required="false" inheritable="false" ref="C680" default="no"/>
      <scm:attributeUse required="false" inheritable="false" ref="C681"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C682"
                        default="shallow-skip"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C683"
                        default="use-last"/>
      <scm:attributeUse required="false" inheritable="false" ref="C684"/>
      <scm:attributeUse required="false" inheritable="false" ref="C685"/>
      <scm:attributeUse required="false" inheritable="false" ref="C686"/>
      <scm:attributeUse required="false" inheritable="false" ref="C687"/>
      <scm:attributeUse required="false" inheritable="false" ref="C688"/>
      <scm:attributeUse required="false" inheritable="false" ref="C689"/>
      <scm:attributeUse required="false" inheritable="false" ref="C690"/>
      <scm:attributeUse required="false" inheritable="false" ref="C691"/>
      <scm:attributeUse required="false" inheritable="false" ref="C692"/>
      <scm:attributeUse required="false" inheritable="false" ref="C693"/>
      <scm:attributeUse required="false" inheritable="false" ref="C694"/>
      <scm:attributeUse required="false" inheritable="false" ref="C695"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C695"
                  name="_visibility"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C686"
                  name="typed"
                  type="C357"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C575"
                  name="_tunnel"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C383"/>
   <scm:attribute id="C175"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"/>
   <scm:complexType id="C369"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C696"/>
      <scm:attributeUse required="false" inheritable="false" ref="C697"/>
      <scm:attributeUse required="false" inheritable="false" ref="C698" default=""/>
      <scm:attributeUse required="false" inheritable="false" ref="C699"/>
      <scm:attributeUse required="false" inheritable="false" ref="C700"/>
      <scm:attributeUse required="false" inheritable="false" ref="C701"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C418"/>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C447"/>
                  <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C107"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C418" to="1"/>
            <scm:edge term="C107" to="2"/>
            <scm:edge term="C447" to="3"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C107" to="2"/>
            <scm:edge term="C447" to="3"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C107" to="2"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C107" to="2"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@select | @_select)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@regex | @_regex)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C696"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C369"/>
   <scm:attribute id="C608"
                  name="_type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C664"
                  name="package-version"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C506"/>
   <scm:attribute id="C297"
                  name="processContents"
                  type="C264"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C640"
                  name="id"
                  type="#ID"
                  global="false"
                  inheritable="false"
                  containingComplexType="C455"/>
   <scm:attribute id="C42"
                  name="base"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C41"/>
   <scm:attribute id="C579"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C391"/>
   <scm:attribute id="C691"
                  name="_on-multiple-match"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:wildcard id="C292"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C32"
                  name="_default-collation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:wildcard id="C334"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C660"
                  name="_validation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C502"/>
   <scm:attribute id="C647"
                  name="validation"
                  type="C94"
                  global="false"
                  inheritable="false"
                  containingComplexType="C451"/>
   <scm:complexType id="C446"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C702"/>
      <scm:attributeUse required="false" inheritable="false" ref="C703"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C416"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C416" to="2"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C416" to="2"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $e in subsequence(xsl:sort, 2) satisfies empty($e/(@stable | @_stable))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@select | @_select)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C702"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C446"/>
   <scm:attribute id="C9"
                  name="test"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C7"/>
   <scm:complexType id="C411"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C704"/>
      <scm:attributeUse required="false" inheritable="false" ref="C705"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C382"/>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C460"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C460" to="2"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C382" to="3"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C460" to="2"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C382" to="3"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@select | @_select)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C605"
                  name="_streamable"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C662"
                  name="schemaLocation"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C459"/>
   <scm:wildcard id="C145"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/1999/XSL/Transform"/>
   <scm:wildcard id="C177"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C221"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C219"/>
   <scm:complexType id="C454"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C706"/>
      <scm:attributeUse required="false" inheritable="false" ref="C707"/>
      <scm:attributeUse required="false" inheritable="false" ref="C708"/>
      <scm:attributeUse required="false" inheritable="false" ref="C709"/>
      <scm:attributeUse required="false" inheritable="false" ref="C710"/>
      <scm:attributeUse required="false" inheritable="false" ref="C711"/>
      <scm:attributeUse required="false" inheritable="false" ref="C712"/>
      <scm:attributeUse required="false" inheritable="false" ref="C713"/>
      <scm:attributeUse required="false" inheritable="false" ref="C714"/>
      <scm:attributeUse required="false" inheritable="false" ref="C715"/>
      <scm:attributeUse required="false" inheritable="false" ref="C716"/>
      <scm:attributeUse required="false" inheritable="false" ref="C717"/>
      <scm:attributeUse required="false" inheritable="false" ref="C718"/>
      <scm:attributeUse required="false" inheritable="false" ref="C719"/>
      <scm:attributeUse required="false" inheritable="false" ref="C720"/>
      <scm:attributeUse required="false" inheritable="false" ref="C721"/>
      <scm:attributeUse required="false" inheritable="false" ref="C722"/>
      <scm:attributeUse required="false" inheritable="false" ref="C723"/>
      <scm:attributeUse required="false" inheritable="false" ref="C724"/>
      <scm:attributeUse required="false" inheritable="false" ref="C725"/>
      <scm:attributeUse required="false" inheritable="false" ref="C726"/>
      <scm:attributeUse required="false" inheritable="false" ref="C727"/>
      <scm:attributeUse required="false" inheritable="false" ref="C728"/>
      <scm:attributeUse required="false" inheritable="false" ref="C729"/>
      <scm:attributeUse required="false" inheritable="false" ref="C730"/>
      <scm:attributeUse required="false" inheritable="false" ref="C731"/>
      <scm:attributeUse required="false" inheritable="false" ref="C732"/>
      <scm:attributeUse required="false" inheritable="false" ref="C733"/>
      <scm:attributeUse required="false" inheritable="false" ref="C734"/>
      <scm:attributeUse required="false" inheritable="false" ref="C735"/>
      <scm:attributeUse required="false" inheritable="false" ref="C736"/>
      <scm:attributeUse required="false" inheritable="false" ref="C737"/>
      <scm:attributeUse required="false" inheritable="false" ref="C738"/>
      <scm:attributeUse required="false" inheritable="false" ref="C739"/>
      <scm:attributeUse required="false" inheritable="false" ref="C740"/>
      <scm:attributeUse required="false" inheritable="false" ref="C741"/>
      <scm:attributeUse required="false" inheritable="false" ref="C742"/>
      <scm:attributeUse required="false" inheritable="false" ref="C743"/>
      <scm:attributeUse required="false" inheritable="false" ref="C744"/>
      <scm:attributeUse required="false" inheritable="false" ref="C745"/>
      <scm:attributeUse required="false" inheritable="false" ref="C746"/>
      <scm:attributeUse required="false" inheritable="false" ref="C747"/>
      <scm:attributeUse required="false" inheritable="false" ref="C748"/>
      <scm:attributeUse required="false" inheritable="false" ref="C749"/>
      <scm:attributeUse required="false" inheritable="false" ref="C750"/>
      <scm:attributeUse required="false" inheritable="false" ref="C751"/>
      <scm:attributeUse required="false" inheritable="false" ref="C752"/>
      <scm:attributeUse required="false" inheritable="false" ref="C753"/>
      <scm:attributeUse required="false" inheritable="false" ref="C754"/>
      <scm:attributeUse required="false" inheritable="false" ref="C755"/>
      <scm:attributeUse required="false" inheritable="false" ref="C756"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@type | @_type) and exists(@validation | @_validation))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C749"
                  name="_normalization-form"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C724"
                  name="media-type"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C731"
                  name="use-character-maps"
                  type="C254"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C747"
                  name="_item-separator"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C735"
                  name="_type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C728"
                  name="standalone"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C718"
                  name="escape-uri-attributes"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C737"
                  name="_method"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C755"
                  name="_use-character-maps"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C714"
                  name="cdata-section-elements"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C750"
                  name="_omit-xml-declaration"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C753"
                  name="_suppress-indentation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C713"
                  name="byte-order-mark"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C730"
                  name="undeclare-prefixes"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C729"
                  name="suppress-indentation"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C739"
                  name="_cdata-section-elements"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C751"
                  name="_parameter-document"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C746"
                  name="_indent"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C661"
                  name="namespace"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C459"/>
   <scm:attribute id="C616"
                  name="finalDefault"
                  type="C364"
                  global="false"
                  inheritable="false"
                  containingComplexType="C475"/>
   <scm:attribute id="C719"
                  name="html-version"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:complexType id="C466"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C757"/>
      <scm:attributeUse required="false" inheritable="false" ref="C758"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@href | @_href)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C758"
                  name="_href"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C466"/>
   <scm:attribute id="C757"
                  name="href"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C466"/>
   <scm:attribute id="C589"
                  name="_copy-namespaces"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C410"/>
   <scm:attribute id="C176"
                  name="ref"
                  type="#QName"
                  global="false"
                  inheritable="false"/>
   <scm:complexType id="C497"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C759"/>
      <scm:attributeUse required="false" inheritable="false" ref="C760"/>
      <scm:attributeUse required="false" inheritable="false" ref="C761"/>
      <scm:attributeUse required="false" inheritable="false" ref="C762"/>
      <scm:attributeUse required="false" inheritable="false" ref="C763"/>
      <scm:attributeUse required="false" inheritable="false" ref="C764"/>
      <scm:attributeUse required="false" inheritable="false" ref="C765"/>
      <scm:attributeUse required="false" inheritable="false" ref="C766"/>
      <scm:attributeUse required="false" inheritable="false" ref="C767"/>
      <scm:attributeUse required="false" inheritable="false" ref="C768"/>
      <scm:attributeUse required="false" inheritable="false" ref="C769"/>
      <scm:attributeUse required="false" inheritable="false" ref="C770"/>
      <scm:attributeUse required="false" inheritable="false" ref="C771"/>
      <scm:attributeUse required="false" inheritable="false" ref="C772"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C426"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C107"/>
         </scm:choice>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C426" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C426" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@xpath | @_xpath)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C762"
                  name="context-item"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C761"
                  name="base-uri"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C769"
                  name="_context-item"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:complexType id="C399"
                    base="C95"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C773"/>
      <scm:attributeUse required="false" inheritable="false" ref="C774"/>
      <scm:attributeUse required="false" inheritable="false" ref="C775"/>
      <scm:attributeUse required="false" inheritable="false" ref="C776"/>
      <scm:attributeUse required="false" inheritable="false" ref="C777"/>
      <scm:attributeUse required="false" inheritable="false" ref="C778"/>
      <scm:attributeUse required="false" inheritable="false" ref="C779"/>
      <scm:attributeUse required="false" inheritable="false" ref="C780"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="if (normalize-space(@static) = ('yes', 'true', '1'))                             then (exists(@_visibility) or normalize-space(@visibility) = ('', 'private', 'final'))                             else true()"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="if (normalize-space(@static) = ('yes', 'true', '1'))                            then (empty((*, text())) and exists(@select | @_select))                             else true()"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C779"
                  name="_visibility"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C399"/>
   <scm:attribute id="C774"
                  name="as"
                  type="C78"
                  global="false"
                  inheritable="false"
                  containingComplexType="C399"/>
   <scm:attribute id="C780"
                  name="_static"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C399"/>
   <scm:attribute id="C568"
                  name="as"
                  type="C78"
                  global="false"
                  inheritable="false"
                  containingComplexType="C383"/>
   <scm:attribute id="C224"
                  name="type"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C223"/>
   <scm:attribute id="C617"
                  name="blockDefault"
                  type="C268"
                  global="false"
                  inheritable="false"
                  containingComplexType="C475"/>
   <scm:complexType id="C490"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="true" inheritable="false" ref="C781"/>
      <scm:attributeUse required="false" inheritable="false" ref="C782"/>
      <scm:attributeUse required="false" inheritable="false" ref="C783"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C783"
                  name="system"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C490"/>
   <scm:attribute id="C666"
                  name="_package-version"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C506"/>
   <scm:wildcard id="C363"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C26"
                  name="default-validation"
                  type="C93"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:key id="C480"
            name="attributeGroup"
            targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:selector xmlns:xs="http://www.w3.org/2001/XMLSchema"
                    xpath="xs:attributeGroup"
                    defaultNamespace=""/>
      <scm:field xmlns:xs="http://www.w3.org/2001/XMLSchema"
                 xpath="@name"
                 defaultNamespace=""/>
   </scm:key>
   <scm:complexType id="C420"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="true" inheritable="false" ref="C784"/>
      <scm:attributeUse required="false" inheritable="false" ref="C785"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C785"
                  name="xpathDefaultNamespace"
                  type="C288"
                  global="false"
                  inheritable="false"
                  containingComplexType="C420"/>
   <scm:attribute id="C784"
                  name="xpath"
                  type="C290"
                  global="false"
                  inheritable="false"
                  containingComplexType="C420"/>
   <scm:attribute id="C697"
                  name="regex"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C369"/>
   <scm:complexType id="C422"
                    base="C309"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C786"/>
      <scm:attributeWildcard ref="C787"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C786"
                  name="value"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C422"/>
   <scm:wildcard id="C787"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:complexType id="C488"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C426"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C426" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C426" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C174"
                  name="minOccurs"
                  type="#nonNegativeInteger"
                  global="false"
                  inheritable="false"/>
   <scm:complexType id="C453"
                    base="C95"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C788"/>
      <scm:attributeUse required="false" inheritable="false" ref="C789"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C788"
                  name="errors"
                  type="#token"
                  global="false"
                  inheritable="false"
                  containingComplexType="C453"/>
   <scm:attribute id="C775"
                  name="visibility"
                  type="C184"
                  global="false"
                  inheritable="false"
                  containingComplexType="C399"/>
   <scm:attribute id="C607"
                  name="_sort-before-merge"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C528"
                  name="expand-text"
                  type="C259"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C716"
                  name="doctype-system"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C205"
                  name="value"
                  type="#anySimpleType"
                  global="false"
                  inheritable="false"
                  containingComplexType="C191"/>
   <scm:attribute id="C744"
                  name="_html-version"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C789"
                  name="_errors"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C453"/>
   <scm:attribute id="C680"
                  name="streamable"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:complexType id="C377"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C790"/>
      <scm:attributeUse required="false" inheritable="false" ref="C791"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@elements | @_elements)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C634"
                  name="_stable"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:attribute id="C765"
                  name="with-params"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C613"
                  name="_mode"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C423"/>
   <scm:attribute id="C658"
                  name="validation"
                  type="C94"
                  global="false"
                  inheritable="false"
                  containingComplexType="C502"/>
   <scm:attribute id="C620"
                  name="defaultAttributes"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C475"/>
   <scm:attribute id="C682"
                  name="on-no-match"
                  type="C6"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C791"
                  name="_elements"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C377"/>
   <scm:attribute id="C694"
                  name="_typed"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C742"
                  name="_encoding"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:complexType id="C472"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="true" inheritable="false" ref="C792"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C792"
                  name="schemaLocation"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C472"/>
   <scm:complexType id="C429"
                    base="C95"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C793"/>
      <scm:attributeUse required="false" inheritable="false" ref="C794"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C230"
                  name="inheritable"
                  type="#boolean"
                  global="false"
                  inheritable="false"
                  containingComplexType="C223"/>
   <scm:attribute id="C635"
                  name="id"
                  type="#ID"
                  global="false"
                  inheritable="false"
                  containingComplexType="C397"/>
   <scm:attribute id="C571"
                  name="static"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C383"/>
   <scm:attribute id="C638"
                  name="_input-type-annotations"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C397"/>
   <scm:attribute id="C708"
                  name="type"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C527"
                  name="default-validation"
                  type="C93"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C623"
                  name="lang"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:attribute id="C34"
                  name="_default-validation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:attribute id="C558"
                  name="_format"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C722"
                  name="item-separator"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C16"
                  name="maxOccurs"
                  type="C276"
                  global="false"
                  inheritable="false"
                  containingComplexType="C14"/>
   <scm:attribute id="C529"
                  name="extension-element-prefixes"
                  targetNamespace="http://www.w3.org/1999/XSL/Transform"
                  type="C22"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C763"
                  name="namespace-context"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C339"
                  name="form"
                  type="C245"
                  global="false"
                  inheritable="false"
                  containingComplexType="C317"/>
   <scm:attribute id="C564"
                  name="_grouping-size"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C324"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C316"/>
   <scm:attribute id="C576"
                  name="_static"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C383"/>
   <scm:attribute id="C703"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C446"/>
   <scm:wildcard id="C265"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:complexType id="C445"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C107"/>
                  <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
                     <scm:choice>
                        <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
                           <scm:sequence>
                              <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C140"/>
                              <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C107"/>
                           </scm:sequence>
                        </scm:modelGroupParticle>
                        <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
                           <scm:sequence>
                              <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C102"/>
                              <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C107"/>
                           </scm:sequence>
                        </scm:modelGroupParticle>
                     </scm:choice>
                  </scm:modelGroupParticle>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C107" to="2"/>
            <scm:edge term="C102" to="3"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C107" to="4"/>
            <scm:edge term="C140" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C107" to="2"/>
            <scm:edge term="C102" to="3"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C107" to="5"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C107" to="4"/>
            <scm:edge term="C140" to="1"/>
         </scm:state>
         <scm:state nr="5" final="true">
            <scm:edge term="C107" to="5"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C384"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C795"/>
      <scm:attributeUse required="false" inheritable="false" ref="C796"/>
      <scm:attributeUse required="false" inheritable="false" ref="C797"/>
      <scm:attributeUse required="false" inheritable="false" ref="C798"/>
      <scm:attributeUse required="false" inheritable="false" ref="C799"/>
      <scm:attributeUse required="false" inheritable="false" ref="C800"/>
      <scm:attributeUse required="false" inheritable="false" ref="C801"/>
      <scm:attributeUse required="false" inheritable="false" ref="C802"/>
      <scm:attributeUse required="false" inheritable="false" ref="C803"/>
      <scm:attributeUse required="false" inheritable="false" ref="C804"/>
      <scm:attributeUse required="false" inheritable="false" ref="C805"/>
      <scm:attributeUse required="false" inheritable="false" ref="C806"/>
      <scm:attributeUse required="false" inheritable="false" ref="C807"/>
      <scm:attributeUse required="false" inheritable="false" ref="C808"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C416"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C416" to="2"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C416" to="2"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="if (exists(@collation|@_collation) or exists(@composite|@_composite))                             then (exists(@group-by|@_group-by) or exists(@group-adjacent|@_group-adjacent)) else true()"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $e in subsequence(xsl:sort, 2) satisfies empty($e/(@stable | @_stable))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="count(((@group-by|@_group-by)[1], (@group-adjacent|@_group-adjacent)[1],                              (@group-starting-with|@_group-starting-with)[1], (@group-ending-with|@_group-ending-with)[1])) = 1"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@select | @_select)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C799"
                  name="group-ending-with"
                  type="C155"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C803"
                  name="_group-by"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C798"
                  name="group-starting-with"
                  type="C155"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C797"
                  name="group-adjacent"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C801"
                  name="collation"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C804"
                  name="_group-adjacent"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C807"
                  name="_composite"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C795"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C685"
                  name="warning-on-multiple-match"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C673"
                  name="_match"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C659"
                  name="_type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C502"/>
   <scm:complexType id="C371"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C809"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C45"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C48" to="1"/>
            <scm:edge term="C49" to="1"/>
            <scm:edge term="C50" to="1"/>
            <scm:edge term="C13" to="3"/>
            <scm:edge term="C52" to="1"/>
            <scm:edge term="C53" to="1"/>
            <scm:edge term="C55" to="1"/>
            <scm:edge term="C56" to="1"/>
            <scm:edge term="C57" to="1"/>
            <scm:edge term="C59" to="1"/>
            <scm:edge term="C62" to="1"/>
            <scm:edge term="C64" to="1"/>
            <scm:edge term="C65" to="1"/>
            <scm:edge term="C66" to="2"/>
            <scm:edge term="C67" to="1"/>
            <scm:edge term="C68" to="1"/>
            <scm:edge term="C69" to="1"/>
            <scm:edge term="C72" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C48" to="1"/>
            <scm:edge term="C49" to="1"/>
            <scm:edge term="C50" to="1"/>
            <scm:edge term="C52" to="1"/>
            <scm:edge term="C53" to="1"/>
            <scm:edge term="C55" to="1"/>
            <scm:edge term="C56" to="1"/>
            <scm:edge term="C57" to="1"/>
            <scm:edge term="C59" to="1"/>
            <scm:edge term="C62" to="1"/>
            <scm:edge term="C64" to="1"/>
            <scm:edge term="C65" to="1"/>
            <scm:edge term="C67" to="1"/>
            <scm:edge term="C68" to="1"/>
            <scm:edge term="C69" to="1"/>
            <scm:edge term="C72" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C48" to="1"/>
            <scm:edge term="C49" to="1"/>
            <scm:edge term="C50" to="1"/>
            <scm:edge term="C52" to="1"/>
            <scm:edge term="C53" to="1"/>
            <scm:edge term="C55" to="1"/>
            <scm:edge term="C56" to="1"/>
            <scm:edge term="C57" to="1"/>
            <scm:edge term="C59" to="1"/>
            <scm:edge term="C62" to="1"/>
            <scm:edge term="C64" to="1"/>
            <scm:edge term="C65" to="1"/>
            <scm:edge term="C67" to="1"/>
            <scm:edge term="C68" to="1"/>
            <scm:edge term="C69" to="1"/>
            <scm:edge term="C72" to="1"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C48" to="1"/>
            <scm:edge term="C49" to="1"/>
            <scm:edge term="C50" to="1"/>
            <scm:edge term="C52" to="1"/>
            <scm:edge term="C53" to="1"/>
            <scm:edge term="C55" to="1"/>
            <scm:edge term="C56" to="1"/>
            <scm:edge term="C57" to="1"/>
            <scm:edge term="C59" to="1"/>
            <scm:edge term="C62" to="1"/>
            <scm:edge term="C64" to="1"/>
            <scm:edge term="C65" to="1"/>
            <scm:edge term="C66" to="2"/>
            <scm:edge term="C67" to="1"/>
            <scm:edge term="C68" to="1"/>
            <scm:edge term="C69" to="1"/>
            <scm:edge term="C72" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C809"
                  name="base"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C371"/>
   <scm:attribute id="C553"
                  name="_value"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C274"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C272"/>
   <scm:attribute id="C179"
                  name="base"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C178"/>
   <scm:wildcard id="C275"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:key id="C476"
            name="element"
            targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:selector xmlns:xs="http://www.w3.org/2001/XMLSchema"
                    xpath="xs:element"
                    defaultNamespace=""/>
      <scm:field xmlns:xs="http://www.w3.org/2001/XMLSchema"
                 xpath="@name"
                 defaultNamespace=""/>
   </scm:key>
   <scm:attribute id="C738"
                  name="_byte-order-mark"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:wildcard id="C170"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C650"
                  name="_separator"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C451"/>
   <scm:attribute id="C676"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C806"
                  name="_group-ending-with"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C592"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:key id="C477"
            name="attribute"
            targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:selector xmlns:xs="http://www.w3.org/2001/XMLSchema"
                    xpath="xs:attribute"
                    defaultNamespace=""/>
      <scm:field xmlns:xs="http://www.w3.org/2001/XMLSchema"
                 xpath="@name"
                 defaultNamespace=""/>
   </scm:key>
   <scm:complexType id="C437"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C810"/>
      <scm:attributeUse required="false" inheritable="false" ref="C811"/>
      <scm:attributeUse required="false" inheritable="false" ref="C812"/>
      <scm:attributeUse required="false" inheritable="false" ref="C813"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C811"
                  name="use"
                  type="C360"
                  global="false"
                  inheritable="false"
                  containingComplexType="C437"/>
   <scm:attribute id="C810"
                  name="as"
                  type="C335"
                  global="false"
                  inheritable="false"
                  containingComplexType="C437"/>
   <scm:attribute id="C813"
                  name="_use"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C437"/>
   <scm:attribute id="C584"
                  name="copy-namespaces"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C410"/>
   <scm:complexType id="C415"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C814"/>
      <scm:attributeUse required="false" inheritable="false" ref="C815" default="yes"/>
      <scm:attributeUse required="false" inheritable="false" ref="C816" default="yes"/>
      <scm:attributeUse required="false" inheritable="false" ref="C817" default=""/>
      <scm:attributeUse required="false" inheritable="false" ref="C818"/>
      <scm:attributeUse required="false" inheritable="false" ref="C819"/>
      <scm:attributeUse required="false" inheritable="false" ref="C820"/>
      <scm:attributeUse required="false" inheritable="false" ref="C821"/>
      <scm:attributeUse required="false" inheritable="false" ref="C822"/>
      <scm:attributeUse required="false" inheritable="false" ref="C823"/>
      <scm:attributeUse required="false" inheritable="false" ref="C824"/>
      <scm:attributeUse required="false" inheritable="false" ref="C825"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@type | @_type) and exists(@validation | @_validation))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C818"
                  name="type"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C825"
                  name="_validation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C823"
                  name="_use-attribute-sets"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C821"
                  name="_copy-namespaces"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C820"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C817"
                  name="use-attribute-sets"
                  type="C254"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C815"
                  name="copy-namespaces"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C814"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C636"
                  name="input-type-annotations"
                  type="C79"
                  global="false"
                  inheritable="false"
                  containingComplexType="C397"/>
   <scm:complexType id="C439"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="1" maxOccurs="unbounded" ref="C393"/>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C510"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C393" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C393" to="2"/>
            <scm:edge term="C510" to="3"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C393" to="2"/>
            <scm:edge term="C510" to="3"/>
         </scm:state>
         <scm:state nr="3" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C296"
                  name="notNamespace"
                  type="C255"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C586"
                  name="validation"
                  type="C94"
                  global="false"
                  inheritable="false"
                  containingComplexType="C410"/>
   <scm:attribute id="C556"
                  name="_count"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C782"
                  name="public"
                  type="C271"
                  global="false"
                  inheritable="false"
                  containingComplexType="C490"/>
   <scm:attribute id="C653"
                  name="character"
                  type="C85"
                  global="false"
                  inheritable="false"
                  containingComplexType="C504"/>
   <scm:attribute id="C591"
                  name="_validation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C410"/>
   <scm:complexType id="C386"
                    base="C95"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C826"/>
      <scm:attributeUse required="false" inheritable="false" ref="C827"/>
      <scm:attributeUse required="false" inheritable="false" ref="C828"/>
      <scm:attributeUse required="false" inheritable="false" ref="C829"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence/>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@match | @_match)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C827"
                  name="phase"
                  type="C257"
                  global="false"
                  inheritable="false"
                  containingComplexType="C386"/>
   <scm:attribute id="C225"
                  name="use"
                  type="C304"
                  global="false"
                  inheritable="false"
                  containingComplexType="C223"/>
   <scm:wildcard id="C196"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C533"
                  name="use-attribute-sets"
                  targetNamespace="http://www.w3.org/1999/XSL/Transform"
                  type="C254"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C692"
                  name="_warning-on-no-match"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C648"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C451"/>
   <scm:attribute id="C652"
                  name="_validation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C451"/>
   <scm:complexType id="C406"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C830" default="yes"/>
      <scm:attributeUse required="false" inheritable="false" ref="C831"/>
      <scm:attributeUse required="false" inheritable="false" ref="C832"/>
      <scm:attributeUse required="false" inheritable="false" ref="C833"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C452"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
                     <scm:choice>
                        <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C452"/>
                        <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C107"/>
                     </scm:choice>
                  </scm:modelGroupParticle>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C452" to="2"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C452" to="2"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C452" to="3"/>
            <scm:edge term="C107" to="3"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C452" to="3"/>
            <scm:edge term="C107" to="3"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C830"
                  name="rollback-output"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C406"/>
   <scm:attribute id="C833"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C406"/>
   <scm:attribute id="C588"
                  name="_copy-accumulators"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C410"/>
   <scm:wildcard id="C12"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C727"
                  name="parameter-document"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C570"
                  name="tunnel"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C383"/>
   <scm:attribute id="C166"
                  name="_disable-output-escaping"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C162"/>
   <scm:element id="C566"
                name="extension"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C291"
                global="false"
                containingComplexType="C467"
                nillable="false"
                abstract="false"/>
   <scm:attribute id="C805"
                  name="_group-starting-with"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:element id="C70"
                name="attribute"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C223"
                global="false"
                nillable="false"
                abstract="false"/>
   <scm:wildcard id="C280"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C678"
                  name="_visibility"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C732"
                  name="output-version"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C348"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C347"/>
   <scm:element id="C54"
                name="group"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C278"
                global="false"
                nillable="false"
                abstract="false"/>
   <scm:complexType id="C435"
                    base="#anyType"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C834"/>
      <scm:attributeUse required="false" inheritable="false" ref="C514"/>
      <scm:attributeWildcard ref="C835"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:sequence>
            <scm:elementWildcard minOccurs="1" maxOccurs="1" ref="C836"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C836" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C836" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C834"
                  name="source"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C435"/>
   <scm:wildcard id="C836" processContents="lax" constraint="any"/>
   <scm:element id="C21"
                name="group"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C837"
                global="false"
                nillable="false"
                abstract="false"/>
   <scm:element id="C63"
                name="assert"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C7"
                global="false"
                nillable="false"
                abstract="false"/>
   <scm:complexType id="C432"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C838"/>
      <scm:attributeUse required="false" inheritable="false" ref="C839"/>
      <scm:attributeUse required="false" inheritable="false" ref="C840"/>
      <scm:attributeUse required="false" inheritable="false" ref="C841"/>
      <scm:attributeUse required="false" inheritable="false" ref="C842"/>
      <scm:attributeUse required="false" inheritable="false" ref="C843"/>
      <scm:attributeUse required="false" inheritable="false" ref="C844"/>
      <scm:attributeUse required="false" inheritable="false" ref="C845"/>
      <scm:attributeUse required="false" inheritable="false" ref="C846"/>
      <scm:attributeUse required="false" inheritable="false" ref="C847"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C493"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C374"/>
            <scm:elementWildcard minOccurs="1" maxOccurs="1" ref="C848"/>
         </scm:choice>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C376" to="1"/>
            <scm:edge term="C491" to="1"/>
            <scm:edge term="C505" to="1"/>
            <scm:edge term="C508" to="1"/>
            <scm:edge term="C390" to="1"/>
            <scm:edge term="C486" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C412" to="1"/>
            <scm:edge term="C374" to="1"/>
            <scm:edge term="C461" to="1"/>
            <scm:edge term="C465" to="1"/>
            <scm:edge term="C468" to="1"/>
            <scm:edge term="C463" to="1"/>
            <scm:edge term="C483" to="1"/>
            <scm:edge term="C380" to="1"/>
            <scm:edge term="C449" to="1"/>
            <scm:edge term="C408" to="1"/>
            <scm:edge term="C382" to="1"/>
            <scm:edge term="C372" to="1"/>
            <scm:edge term="C493" to="1"/>
            <scm:edge term="C424" to="1"/>
            <scm:edge term="C848" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C376" to="1"/>
            <scm:edge term="C491" to="1"/>
            <scm:edge term="C505" to="1"/>
            <scm:edge term="C508" to="1"/>
            <scm:edge term="C390" to="1"/>
            <scm:edge term="C486" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C412" to="1"/>
            <scm:edge term="C374" to="1"/>
            <scm:edge term="C461" to="1"/>
            <scm:edge term="C465" to="1"/>
            <scm:edge term="C468" to="1"/>
            <scm:edge term="C463" to="1"/>
            <scm:edge term="C483" to="1"/>
            <scm:edge term="C380" to="1"/>
            <scm:edge term="C449" to="1"/>
            <scm:edge term="C408" to="1"/>
            <scm:edge term="C382" to="1"/>
            <scm:edge term="C372" to="1"/>
            <scm:edge term="C493" to="1"/>
            <scm:edge term="C424" to="1"/>
            <scm:edge term="C848" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C843"
                  name="_declared-modes"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C432"/>
   <scm:attribute id="C844"
                  name="_id"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C432"/>
   <scm:attribute id="C847"
                  name="_input-type-annotations"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C432"/>
   <scm:attribute id="C840"
                  name="name"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C432"/>
   <scm:attribute id="C846"
                  name="_package-version"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C432"/>
   <scm:attribute id="C839"
                  name="id"
                  type="#ID"
                  global="false"
                  inheritable="false"
                  containingComplexType="C432"/>
   <scm:attribute id="C842"
                  name="input-type-annotations"
                  type="C79"
                  global="false"
                  inheritable="false"
                  containingComplexType="C432"/>
   <scm:attribute id="C838"
                  name="declared-modes"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C432"/>
   <scm:element id="C328"
                name="alternative"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C86"
                global="false"
                containingComplexType="C316"
                nillable="false"
                abstract="false"/>
   <scm:complexType id="C444"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C463"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C483"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C115"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C382"/>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C491"/>
         </scm:choice>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C382" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C463" to="1"/>
            <scm:edge term="C483" to="1"/>
            <scm:edge term="C491" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C382" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C463" to="1"/>
            <scm:edge term="C483" to="1"/>
            <scm:edge term="C491" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:element id="C285"
                name="all"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C849"
                global="false"
                containingComplexType="C282"
                nillable="false"
                abstract="false"/>
   <scm:complexType id="C849"
                    base="C14"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C850"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C19"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C20"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C21"/>
               </scm:choice>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C21" to="1"/>
            <scm:edge term="C13" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C21" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C19" to="1"/>
            <scm:edge term="C20" to="1"/>
            <scm:edge term="C21" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:wildcard id="C850"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C313"
                  name="version"
                  type="#decimal"
                  global="false"
                  inheritable="false"
                  containingComplexType="C311"/>
   <scm:wildcard id="C99"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/1999/XSL/Transform"/>
   <scm:attribute id="C247"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C246"/>
   <scm:complexType id="C381"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C851"/>
      <scm:attributeUse required="false" inheritable="false" ref="C852"/>
      <scm:attributeUse required="false" inheritable="false" ref="C853"/>
      <scm:attributeUse required="false" inheritable="false" ref="C854"/>
      <scm:attributeUse required="false" inheritable="false" ref="C855"/>
      <scm:attributeUse required="false" inheritable="false" ref="C856"/>
      <scm:attributeUse required="false" inheritable="false" ref="C857"/>
      <scm:attributeUse required="false" inheritable="false" ref="C858"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:elementParticle minOccurs="1" maxOccurs="unbounded" ref="C385"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C385" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C385" to="2"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C385" to="2"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C855"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C381"/>
   <scm:attribute id="C856"
                  name="_initial-value"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C381"/>
   <scm:attribute id="C858"
                  name="_streamable"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C381"/>
   <scm:attribute id="C711"
                  name="allow-duplicate-names"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C828"
                  name="_match"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C386"/>
   <scm:wildcard id="C144"
                 processContents="lax"
                 constraint="enumeration"
                 namespaces="##local"/>
   <scm:element id="C60"
                name="attributeGroup"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C80"
                global="false"
                nillable="false"
                abstract="false"/>
   <scm:element id="C286"
                name="choice"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C333"
                global="false"
                containingComplexType="C282"
                nillable="false"
                abstract="false"/>
   <scm:attribute id="C793"
                  name="name"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C429"/>
   <scm:attribute id="C852"
                  name="initial-value"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C381"/>
   <scm:element id="C90"
                name="complexType"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C358"
                global="false"
                containingComplexType="C86"
                nillable="false"
                abstract="false"/>
   <scm:attribute id="C279"
                  name="ref"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C278"/>
   <scm:complexType id="C428"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C859"/>
      <scm:attributeUse required="false" inheritable="false" ref="C860"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@test | @_test)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C859"
                  name="test"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C428"/>
   <scm:element id="C19"
                name="element"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C338"
                global="false"
                nillable="false"
                abstract="false"/>
   <scm:attribute id="C353"
                  name="defaultAttributesApply"
                  type="#boolean"
                  global="false"
                  inheritable="false"
                  containingComplexType="C347"/>
   <scm:complexType id="C433"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C861"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:choice>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C862"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C863"/>
               </scm:choice>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C863" to="1"/>
            <scm:edge term="C13" to="2"/>
            <scm:edge term="C862" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
         <scm:state nr="2">
            <scm:edge term="C863" to="1"/>
            <scm:edge term="C862" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:element id="C863"
                name="extension"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C178"
                global="false"
                containingComplexType="C433"
                nillable="false"
                abstract="false"/>
   <scm:attribute id="C572"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C383"/>
   <scm:attribute id="C665"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C506"/>
   <scm:attribute id="C819"
                  name="validation"
                  type="C94"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C808"
                  name="_collation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C38"
                  name="_use-when"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:wildcard id="C222"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:complexType id="C389"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C864"/>
      <scm:attributeUse required="false" inheritable="false" ref="C865"/>
      <scm:attributeUse required="false" inheritable="false" ref="C866"/>
      <scm:attributeUse required="false" inheritable="false" ref="C867"/>
      <scm:attributeUse required="false" inheritable="false" ref="C868"/>
      <scm:attributeUse required="false" inheritable="false" ref="C869"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@test | @_test)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C867"
                  name="_test"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C389"/>
   <scm:attribute id="C868"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C389"/>
   <scm:attribute id="C869"
                  name="_error-code"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C389"/>
   <scm:attribute id="C631"
                  name="_order"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:attribute id="C693"
                  name="_warning-on-multiple-match"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C760"
                  name="as"
                  type="C78"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C705"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C411"/>
   <scm:wildcard id="C327"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C698"
                  name="flags"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C369"/>
   <scm:complexType id="C401"
                    base="C289"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="true" inheritable="false" ref="C870"/>
      <scm:attributeUse required="false" inheritable="false" ref="C871"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:choice>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1" ref="C517"/>
         </scm:choice>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C414" to="1"/>
            <scm:edge term="C507" to="1"/>
            <scm:edge term="C438" to="1"/>
            <scm:edge term="C13" to="1"/>
            <scm:edge term="C456" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C414" to="1"/>
            <scm:edge term="C507" to="1"/>
            <scm:edge term="C438" to="1"/>
            <scm:edge term="C13" to="1"/>
            <scm:edge term="C456" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C871"
                  name="id"
                  type="#ID"
                  global="false"
                  inheritable="false"
                  containingComplexType="C401"/>
   <scm:attribute id="C715"
                  name="doctype-public"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C812"
                  name="_as"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C437"/>
   <scm:complexType id="C509"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C872"/>
      <scm:attributeUse required="false" inheritable="false" ref="C873"/>
      <scm:attributeUse required="false" inheritable="false" ref="C874"/>
      <scm:attributeUse required="false" inheritable="false" ref="C875"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C474"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C474" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@schema-location | @_schema-location) and exists(xs:schema))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C874"
                  name="_namespace"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C509"/>
   <scm:attribute id="C873"
                  name="schema-location"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C509"/>
   <scm:attribute id="C875"
                  name="_schema-location"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C509"/>
   <scm:element id="C862"
                name="restriction"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C217"
                global="false"
                containingComplexType="C433"
                nillable="false"
                abstract="false"/>
   <scm:complexType id="C379"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C876"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="1" maxOccurs="unbounded" ref="C416"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C416" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="2"/>
            <scm:edge term="C102" to="2"/>
            <scm:edge term="C103" to="2"/>
            <scm:edge term="C104" to="2"/>
            <scm:edge term="C105" to="2"/>
            <scm:edge term="C106" to="2"/>
            <scm:edge term="C107" to="2"/>
            <scm:edge term="C108" to="2"/>
            <scm:edge term="C109" to="2"/>
            <scm:edge term="C110" to="2"/>
            <scm:edge term="C111" to="2"/>
            <scm:edge term="C112" to="2"/>
            <scm:edge term="C113" to="2"/>
            <scm:edge term="C114" to="2"/>
            <scm:edge term="C115" to="2"/>
            <scm:edge term="C116" to="2"/>
            <scm:edge term="C117" to="2"/>
            <scm:edge term="C118" to="2"/>
            <scm:edge term="C119" to="2"/>
            <scm:edge term="C120" to="2"/>
            <scm:edge term="C121" to="2"/>
            <scm:edge term="C122" to="2"/>
            <scm:edge term="C123" to="2"/>
            <scm:edge term="C124" to="2"/>
            <scm:edge term="C125" to="2"/>
            <scm:edge term="C416" to="3"/>
            <scm:edge term="C126" to="2"/>
            <scm:edge term="C127" to="2"/>
            <scm:edge term="C128" to="2"/>
            <scm:edge term="C129" to="2"/>
            <scm:edge term="C130" to="2"/>
            <scm:edge term="C131" to="2"/>
            <scm:edge term="C132" to="2"/>
            <scm:edge term="C133" to="2"/>
            <scm:edge term="C134" to="2"/>
            <scm:edge term="C135" to="2"/>
            <scm:edge term="C136" to="2"/>
            <scm:edge term="C137" to="2"/>
            <scm:edge term="C138" to="2"/>
            <scm:edge term="C139" to="2"/>
            <scm:edge term="C140" to="2"/>
            <scm:edge term="C141" to="2"/>
            <scm:edge term="C142" to="2"/>
            <scm:edge term="C143" to="2"/>
            <scm:edge term="C144" to="2"/>
            <scm:edge term="C145" to="2"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C101" to="2"/>
            <scm:edge term="C102" to="2"/>
            <scm:edge term="C103" to="2"/>
            <scm:edge term="C104" to="2"/>
            <scm:edge term="C105" to="2"/>
            <scm:edge term="C106" to="2"/>
            <scm:edge term="C107" to="2"/>
            <scm:edge term="C108" to="2"/>
            <scm:edge term="C109" to="2"/>
            <scm:edge term="C110" to="2"/>
            <scm:edge term="C111" to="2"/>
            <scm:edge term="C112" to="2"/>
            <scm:edge term="C113" to="2"/>
            <scm:edge term="C114" to="2"/>
            <scm:edge term="C115" to="2"/>
            <scm:edge term="C116" to="2"/>
            <scm:edge term="C117" to="2"/>
            <scm:edge term="C118" to="2"/>
            <scm:edge term="C119" to="2"/>
            <scm:edge term="C120" to="2"/>
            <scm:edge term="C121" to="2"/>
            <scm:edge term="C122" to="2"/>
            <scm:edge term="C123" to="2"/>
            <scm:edge term="C124" to="2"/>
            <scm:edge term="C125" to="2"/>
            <scm:edge term="C126" to="2"/>
            <scm:edge term="C127" to="2"/>
            <scm:edge term="C128" to="2"/>
            <scm:edge term="C129" to="2"/>
            <scm:edge term="C130" to="2"/>
            <scm:edge term="C131" to="2"/>
            <scm:edge term="C132" to="2"/>
            <scm:edge term="C133" to="2"/>
            <scm:edge term="C134" to="2"/>
            <scm:edge term="C135" to="2"/>
            <scm:edge term="C136" to="2"/>
            <scm:edge term="C137" to="2"/>
            <scm:edge term="C138" to="2"/>
            <scm:edge term="C139" to="2"/>
            <scm:edge term="C140" to="2"/>
            <scm:edge term="C141" to="2"/>
            <scm:edge term="C142" to="2"/>
            <scm:edge term="C143" to="2"/>
            <scm:edge term="C144" to="2"/>
            <scm:edge term="C145" to="2"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C101" to="2"/>
            <scm:edge term="C102" to="2"/>
            <scm:edge term="C103" to="2"/>
            <scm:edge term="C104" to="2"/>
            <scm:edge term="C105" to="2"/>
            <scm:edge term="C106" to="2"/>
            <scm:edge term="C107" to="2"/>
            <scm:edge term="C108" to="2"/>
            <scm:edge term="C109" to="2"/>
            <scm:edge term="C110" to="2"/>
            <scm:edge term="C111" to="2"/>
            <scm:edge term="C112" to="2"/>
            <scm:edge term="C113" to="2"/>
            <scm:edge term="C114" to="2"/>
            <scm:edge term="C115" to="2"/>
            <scm:edge term="C116" to="2"/>
            <scm:edge term="C117" to="2"/>
            <scm:edge term="C118" to="2"/>
            <scm:edge term="C119" to="2"/>
            <scm:edge term="C120" to="2"/>
            <scm:edge term="C121" to="2"/>
            <scm:edge term="C122" to="2"/>
            <scm:edge term="C123" to="2"/>
            <scm:edge term="C124" to="2"/>
            <scm:edge term="C125" to="2"/>
            <scm:edge term="C416" to="3"/>
            <scm:edge term="C126" to="2"/>
            <scm:edge term="C127" to="2"/>
            <scm:edge term="C128" to="2"/>
            <scm:edge term="C129" to="2"/>
            <scm:edge term="C130" to="2"/>
            <scm:edge term="C131" to="2"/>
            <scm:edge term="C132" to="2"/>
            <scm:edge term="C133" to="2"/>
            <scm:edge term="C134" to="2"/>
            <scm:edge term="C135" to="2"/>
            <scm:edge term="C136" to="2"/>
            <scm:edge term="C137" to="2"/>
            <scm:edge term="C138" to="2"/>
            <scm:edge term="C139" to="2"/>
            <scm:edge term="C140" to="2"/>
            <scm:edge term="C141" to="2"/>
            <scm:edge term="C142" to="2"/>
            <scm:edge term="C143" to="2"/>
            <scm:edge term="C144" to="2"/>
            <scm:edge term="C145" to="2"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $e in subsequence(xsl:sort, 2) satisfies empty($e/(@stable | @_stable))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C876"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C379"/>
   <scm:complexType id="C470"
                    base="C191"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C877"/>
      <scm:attributeUse required="false" inheritable="false" ref="C193" default="false"/>
      <scm:attributeWildcard ref="C878"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C877"
                  name="value"
                  type="C343"
                  global="false"
                  inheritable="false"
                  containingComplexType="C470"/>
   <scm:element id="C565"
                name="restriction"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C195"
                global="false"
                containingComplexType="C467"
                nillable="false"
                abstract="false"/>
   <scm:element id="C287"
                name="sequence"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C333"
                global="false"
                containingComplexType="C282"
                nillable="false"
                abstract="false"/>
   <scm:wildcard id="C284"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C629"
                  name="_lang"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:attribute id="C864"
                  name="test"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C389"/>
   <scm:element id="C66"
                name="simpleType"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C198"
                global="false"
                nillable="false"
                abstract="false"/>
   <scm:attribute id="C637"
                  name="_id"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C397"/>
   <scm:attribute id="C559"
                  name="_lang"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C776"
                  name="static"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C399"/>
   <scm:attribute id="C720"
                  name="include-content-type"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C322"
                  name="abstract"
                  type="#boolean"
                  global="false"
                  inheritable="false"
                  containingComplexType="C317"/>
   <scm:attribute id="C872"
                  name="namespace"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C509"/>
   <scm:wildcard id="C848"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/1999/XSL/Transform"/>
   <scm:attribute id="C824"
                  name="_type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C320"
                  name="type"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C317"/>
   <scm:complexType id="C413"
                    base="C23"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C879"/>
      <scm:attributeUse required="false" inheritable="false" ref="C880"/>
      <scm:attributeUse required="false" inheritable="false" ref="C881"/>
      <scm:attributeUse required="false" inheritable="false" ref="C882"/>
      <scm:attributeUse required="false" inheritable="false" ref="C883"/>
      <scm:attributeUse required="false" inheritable="false" ref="C884"/>
      <scm:attributeUse required="false" inheritable="false" ref="C885"/>
      <scm:attributeUse required="false" inheritable="false" ref="C886"/>
      <scm:attributeUse required="false" inheritable="false" ref="C887"/>
      <scm:attributeUse required="false" inheritable="false" ref="C888"/>
      <scm:attributeUse required="false" inheritable="false" ref="C889"/>
      <scm:attributeUse required="false" inheritable="false" ref="C890"/>
      <scm:attributeUse required="false" inheritable="false" ref="C891"/>
      <scm:attributeUse required="false" inheritable="false" ref="C892"/>
      <scm:attributeUse required="false" inheritable="false" ref="C893"/>
      <scm:attributeUse required="false" inheritable="false" ref="C894"/>
      <scm:attributeUse required="false" inheritable="false" ref="C895"/>
      <scm:attributeUse required="false" inheritable="false" ref="C896"/>
      <scm:attributeUse required="false" inheritable="false" ref="C897"/>
      <scm:attributeUse required="false" inheritable="false" ref="C898"/>
      <scm:attributeUse required="false" inheritable="false" ref="C899"/>
      <scm:attributeUse required="false" inheritable="false" ref="C900"/>
      <scm:attributeUse required="false" inheritable="false" ref="C901"/>
      <scm:attributeUse required="false" inheritable="false" ref="C902"/>
      <scm:attributeUse required="false" inheritable="false" ref="C903"/>
      <scm:attributeUse required="false" inheritable="false" ref="C904"/>
      <scm:attributeUse required="false" inheritable="false" ref="C905"/>
      <scm:attributeUse required="false" inheritable="false" ref="C906"/>
      <scm:attributeUse required="false" inheritable="false" ref="C907"/>
      <scm:attributeUse required="false" inheritable="false" ref="C908"/>
      <scm:attributeUse required="false" inheritable="false" ref="C909"/>
      <scm:attributeUse required="false" inheritable="false" ref="C910"/>
      <scm:attributeUse required="false" inheritable="false" ref="C911"/>
      <scm:attributeUse required="false" inheritable="false" ref="C912"/>
      <scm:attributeUse required="false" inheritable="false" ref="C913"/>
      <scm:attributeUse required="false" inheritable="false" ref="C914"/>
      <scm:attributeUse required="false" inheritable="false" ref="C915"/>
      <scm:attributeUse required="false" inheritable="false" ref="C916"/>
      <scm:attributeUse required="false" inheritable="false" ref="C917"/>
      <scm:attributeUse required="false" inheritable="false" ref="C918"/>
      <scm:attributeUse required="false" inheritable="false" ref="C919"/>
      <scm:attributeUse required="false" inheritable="false" ref="C920"/>
      <scm:attributeUse required="false" inheritable="false" ref="C921"/>
      <scm:attributeUse required="false" inheritable="false" ref="C922"/>
      <scm:attributeUse required="false" inheritable="false" ref="C923"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C913"
                  name="_indent"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C903"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C884"
                  name="cdata-section-elements"
                  type="C254"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C887"
                  name="encoding"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C880"
                  name="method"
                  type="C252"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C907"
                  name="_doctype-public"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C910"
                  name="_escape-uri-attributes"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C881"
                  name="allow-duplicate-names"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C918"
                  name="_parameter-document"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C898"
                  name="standalone"
                  type="C2"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C917"
                  name="_omit-xml-declaration"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C891"
                  name="indent"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C904"
                  name="_method"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C897"
                  name="parameter-document"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C882"
                  name="build-tree"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C900"
                  name="undeclare-prefixes"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C906"
                  name="_cdata-section-elements"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C914"
                  name="_item-separator"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C921"
                  name="_undeclare-prefixes"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C901"
                  name="use-character-maps"
                  type="C254"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C896"
                  name="omit-xml-declaration"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C909"
                  name="_encoding"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C888"
                  name="escape-uri-attributes"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C920"
                  name="_suppress-indentation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C894"
                  name="media-type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C879"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C893"
                  name="json-node-output-method"
                  type="C252"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C899"
                  name="suppress-indentation"
                  type="C254"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C883"
                  name="byte-order-mark"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C97"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C96"/>
   <scm:attribute id="C603"
                  name="_for-each-source"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C752"
                  name="_standalone"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C915"
                  name="_media-type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:complexType id="C494"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C924"/>
      <scm:attributeUse required="false" inheritable="false" ref="C925"/>
      <scm:attributeUse required="false" inheritable="false" ref="C926"/>
      <scm:attributeUse required="false" inheritable="false" ref="C927"/>
      <scm:attributeUse required="false" inheritable="false" ref="C928"/>
      <scm:attributeUse required="false" inheritable="false" ref="C929"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C927"
                  name="_component"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C494"/>
   <scm:attribute id="C929"
                  name="_visibility"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C494"/>
   <scm:attribute id="C756"
                  name="_output-version"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C349"
                  name="mixed"
                  type="#boolean"
                  global="false"
                  inheritable="false"
                  containingComplexType="C347"/>
   <scm:attribute id="C625"
                  name="order"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:complexType id="C394"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C930"/>
      <scm:attributeUse required="false" inheritable="false" ref="C931"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@test | @_test)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C931"
                  name="_test"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C394"/>
   <scm:attribute id="C930"
                  name="test"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C394"/>
   <scm:attribute id="C704"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C411"/>
   <scm:attribute id="C748"
                  name="_media-type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C657"
                  name="type"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C502"/>
   <scm:attribute id="C563"
                  name="_grouping-separator"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C822"
                  name="_inherit-namespaces"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C709"
                  name="validation"
                  type="C94"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:key id="C478"
            name="type"
            targetNamespace="http://www.w3.org/2001/XMLSchema">
      <scm:selector xmlns:xs="http://www.w3.org/2001/XMLSchema"
                    xpath="xs:complexType|xs:simpleType"
                    defaultNamespace=""/>
      <scm:field xmlns:xs="http://www.w3.org/2001/XMLSchema"
                 xpath="@name"
                 defaultNamespace=""/>
   </scm:key>
   <scm:attribute id="C800"
                  name="composite"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:complexType id="C501"
                    base="#anyType"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C932"/>
      <scm:attributeWildcard ref="C933"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:sequence>
            <scm:elementWildcard minOccurs="1" maxOccurs="1" ref="C934"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C934" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C934" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:wildcard id="C233"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:complexType id="C440"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="1" maxOccurs="unbounded" ref="C402"/>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C407"/>
                  <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C107"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C402" to="1"/>
         </scm:state>
         <scm:state nr="1">
            <scm:edge term="C402" to="2"/>
            <scm:edge term="C407" to="3"/>
         </scm:state>
         <scm:state nr="2">
            <scm:edge term="C402" to="2"/>
            <scm:edge term="C407" to="3"/>
         </scm:state>
         <scm:state nr="3" final="true">
            <scm:edge term="C107" to="4"/>
         </scm:state>
         <scm:state nr="4" final="true">
            <scm:edge term="C107" to="4"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C37"
                  name="_extension-element-prefixes"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:complexType id="C442"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C935"/>
      <scm:attributeUse required="false" inheritable="false" ref="C936"/>
      <scm:attributeUse required="false" inheritable="false" ref="C937"/>
      <scm:attributeUse required="false" inheritable="false" ref="C938"/>
      <scm:attributeUse required="false" inheritable="false" ref="C939"/>
      <scm:attributeUse required="false" inheritable="false" ref="C940"/>
      <scm:attributeUse required="false" inheritable="false" ref="C941"/>
      <scm:attributeUse required="false" inheritable="false" ref="C942"/>
      <scm:attributeUse required="false" inheritable="false" ref="C943"/>
      <scm:attributeUse required="false" inheritable="false" ref="C944"/>
      <scm:attributeUse required="false" inheritable="false" ref="C945"/>
      <scm:attributeUse required="false" inheritable="false" ref="C946"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C939"
                  name="case-order"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C938"
                  name="collation"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C944"
                  name="_collation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C945"
                  name="_case-order"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C936"
                  name="lang"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C942"
                  name="_lang"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C940"
                  name="data-type"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C946"
                  name="_data-type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:complexType id="C441"
                    base="C96"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C947"/>
      <scm:attributeUse required="false" inheritable="false" ref="C948"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@key | @_key)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C947"
                  name="key"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C441"/>
   <scm:attribute id="C948"
                  name="_key"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C441"/>
   <scm:attribute id="C646"
                  name="type"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C451"/>
   <scm:attribute id="C777"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C399"/>
   <scm:attribute id="C857"
                  name="_as"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C381"/>
   <scm:attribute id="C681"
                  name="use-accumulators"
                  type="C180"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C816"
                  name="inherit-namespaces"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C415"/>
   <scm:attribute id="C601"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C781"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C490"/>
   <scm:attribute id="C534"
                  name="use-when"
                  targetNamespace="http://www.w3.org/1999/XSL/Transform"
                  type="C156"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C829"
                  name="_phase"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C386"/>
   <scm:attribute id="C865"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C389"/>
   <scm:complexType id="C425"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C949"/>
      <scm:attributeUse required="false" inheritable="false" ref="C950"/>
      <scm:attributeUse required="false" inheritable="false" ref="C951"/>
      <scm:attributeUse required="false" inheritable="false" ref="C952"/>
      <scm:attributeUse required="false" inheritable="false" ref="C953"/>
      <scm:attributeUse required="false" inheritable="false" ref="C954"/>
      <scm:attributeUse required="false" inheritable="false" ref="C955"/>
      <scm:attributeUse required="false" inheritable="false" ref="C956"/>
      <scm:attributeUse required="false" inheritable="false" ref="C957"/>
      <scm:attributeUse required="false" inheritable="false" ref="C958"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@match | @_match)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C952"
                  name="composite"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C425"/>
   <scm:attribute id="C950"
                  name="match"
                  type="C155"
                  global="false"
                  inheritable="false"
                  containingComplexType="C425"/>
   <scm:attribute id="C949"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C425"/>
   <scm:attribute id="C956"
                  name="_use"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C425"/>
   <scm:attribute id="C951"
                  name="use"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C425"/>
   <scm:attribute id="C655"
                  name="_character"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C504"/>
   <scm:attribute id="C925"
                  name="names"
                  type="C254"
                  global="false"
                  inheritable="false"
                  containingComplexType="C494"/>
   <scm:complexType id="C462"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C959"/>
      <scm:attributeUse required="false" inheritable="false" ref="C960"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@elements | @_elements)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C959"
                  name="elements"
                  type="C0"
                  global="false"
                  inheritable="false"
                  containingComplexType="C462"/>
   <scm:attribute id="C960"
                  name="_elements"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C462"/>
   <scm:attribute id="C890"
                  name="include-content-type"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C919"
                  name="_standalone"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C958"
                  name="_collation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C425"/>
   <scm:attribute id="C581"
                  name="itemType"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C499"/>
   <scm:attribute id="C710"
                  name="method"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C923"
                  name="_version"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C615"
                  name="version"
                  type="#token"
                  global="false"
                  inheritable="false"
                  containingComplexType="C475"/>
   <scm:attribute id="C670"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C362"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C361"/>
   <scm:complexType id="C487"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C961"/>
      <scm:attributeUse required="false" inheritable="false" ref="C962"/>
      <scm:attributeUse required="false" inheritable="false" ref="C963"/>
      <scm:attributeUse required="false" inheritable="false" ref="C964"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C962"
                  name="use"
                  type="C281"
                  global="false"
                  inheritable="false"
                  containingComplexType="C487"/>
   <scm:attribute id="C961"
                  name="as"
                  type="C335"
                  global="false"
                  inheritable="false"
                  containingComplexType="C487"/>
   <scm:attribute id="C964"
                  name="_use"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C487"/>
   <scm:attribute id="C924"
                  name="component"
                  type="C307"
                  global="false"
                  inheritable="false"
                  containingComplexType="C494"/>
   <scm:attribute id="C741"
                  name="_doctype-system"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C905"
                  name="_byte-order-mark"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C922"
                  name="_use-character-maps"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C679"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C841"
                  name="package-version"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C432"/>
   <scm:attribute id="C643"
                  name="name"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C451"/>
   <scm:complexType id="C485"
                    base="C95"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C965"/>
      <scm:attributeUse required="false" inheritable="false" ref="C966" default="no"/>
      <scm:attributeUse required="false" inheritable="false" ref="C967"/>
      <scm:attributeUse required="false" inheritable="false" ref="C968"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C965"
                  name="separator"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C485"/>
   <scm:attribute id="C967"
                  name="_separator"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C485"/>
   <scm:attribute id="C851"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C381"/>
   <scm:attribute id="C736"
                  name="_validation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:wildcard id="C77"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/1999/XSL/Transform"/>
   <scm:attribute id="C562"
                  name="_start-at"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C621"
                  name="xpathDefaultNamespace"
                  type="C288"
                  global="false"
                  inheritable="false"
                  containingComplexType="C475"/>
   <scm:complexType id="C511"
                    base="C190"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C969"/>
      <scm:attributeUse required="false" inheritable="false" ref="C193" default="false"/>
      <scm:attributeWildcard ref="C970"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:wildcard id="C970"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C969"
                  name="value"
                  type="#positiveInteger"
                  global="false"
                  inheritable="false"
                  containingComplexType="C511"/>
   <scm:attribute id="C721"
                  name="indent"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C578"
                  name="use-character-maps"
                  type="C254"
                  global="false"
                  inheritable="false"
                  containingComplexType="C391"/>
   <scm:complexType id="C388"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C971"/>
      <scm:attributeUse required="false" inheritable="false" ref="C972"/>
      <scm:attributeUse required="false" inheritable="false" ref="C973"/>
      <scm:attributeUse required="false" inheritable="false" ref="C974"/>
      <scm:attributeUse required="false" inheritable="false" ref="C975"/>
      <scm:attributeUse required="false" inheritable="false" ref="C976"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@component | @_component)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@names | @_names)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@visibility | @_visibility)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C971"
                  name="component"
                  type="C307"
                  global="false"
                  inheritable="false"
                  containingComplexType="C388"/>
   <scm:attribute id="C976"
                  name="_visibility"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C388"/>
   <scm:attribute id="C974"
                  name="_component"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C388"/>
   <scm:attribute id="C725"
                  name="normalization-form"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C943"
                  name="_order"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C89"
                  name="xpathDefaultNamespace"
                  type="C288"
                  global="false"
                  inheritable="false"
                  containingComplexType="C86"/>
   <scm:attribute id="C226"
                  name="default"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C223"/>
   <scm:attribute id="C651"
                  name="_type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C451"/>
   <scm:attribute id="C11"
                  name="id"
                  type="#ID"
                  global="false"
                  inheritable="false"
                  containingComplexType="C8"/>
   <scm:complexType id="C370"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="true" inheritable="false" ref="C977"/>
      <scm:attributeUse required="false" inheritable="false" ref="C978"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C978"
                  name="xpathDefaultNamespace"
                  type="C288"
                  global="false"
                  inheritable="false"
                  containingComplexType="C370"/>
   <scm:complexType id="C378"
                    base="C191"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="true" inheritable="false" ref="C979"/>
      <scm:attributeUse required="false" inheritable="false" ref="C193" default="false"/>
      <scm:attributeWildcard ref="C980"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C979"
                  name="value"
                  type="C189"
                  global="false"
                  inheritable="false"
                  containingComplexType="C378"/>
   <scm:attribute id="C602"
                  name="_for-each-item"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C642"
                  name="refer"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C404"/>
   <scm:attribute id="C569"
                  name="required"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C383"/>
   <scm:attribute id="C530"
                  name="exclude-result-prefixes"
                  targetNamespace="http://www.w3.org/1999/XSL/Transform"
                  type="C22"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C700"
                  name="_regex"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C369"/>
   <scm:attribute id="C892"
                  name="item-separator"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:complexType id="C448"
                    base="C294"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C981"/>
      <scm:attributeUse required="false" inheritable="false" ref="C295"/>
      <scm:attributeUse required="false" inheritable="false" ref="C296"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C297"
                        default="strict"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C981"
                  name="notQName"
                  type="C213"
                  global="false"
                  inheritable="false"
                  containingComplexType="C448"/>
   <scm:complexType id="C457"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C982"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C66"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C66" to="1"/>
            <scm:edge term="C13" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C66" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C66" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C982"
                  name="memberTypes"
                  type="C345"
                  global="false"
                  inheritable="false"
                  containingComplexType="C457"/>
   <scm:attribute id="C531"
                  name="xpath-default-namespace"
                  targetNamespace="http://www.w3.org/1999/XSL/Transform"
                  type="#anyURI"
                  global="false"
                  inheritable="false"/>
   <scm:attribute id="C860"
                  name="_test"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C428"/>
   <scm:attribute id="C551"
                  name="grouping-separator"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C604"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C684"
                  name="warning-on-no-match"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C796"
                  name="group-by"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C688"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:wildcard id="C878"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C689"
                  name="_streamable"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C972"
                  name="names"
                  type="C254"
                  global="false"
                  inheritable="false"
                  containingComplexType="C388"/>
   <scm:complexType id="C450"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C983"/>
      <scm:attributeUse required="false" inheritable="false" ref="C984"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@href | @_href)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C984"
                  name="_href"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C450"/>
   <scm:attribute id="C983"
                  name="href"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C450"/>
   <scm:attribute id="C600"
                  name="validation"
                  type="C94"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C845"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C432"/>
   <scm:attribute id="C542"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C957"
                  name="_composite"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C425"/>
   <scm:attribute id="C593"
                  name="for-each-item"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C772"
                  name="_with-params"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C574"
                  name="_required"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C383"/>
   <scm:attribute id="C854"
                  name="streamable"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C381"/>
   <scm:attribute id="C550"
                  name="start-at"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C743"
                  name="_escape-uri-attributes"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C764"
                  name="schema-aware"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C690"
                  name="_on-no-match"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:wildcard id="C835"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C88"
                  name="type"
                  type="#QName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C86"/>
   <scm:attribute id="C10"
                  name="xpathDefaultNamespace"
                  type="C288"
                  global="false"
                  inheritable="false"
                  containingComplexType="C7"/>
   <scm:complexType id="C427"
                    base="C95"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C985"/>
      <scm:attributeUse required="false" inheritable="false" ref="C986"/>
      <scm:attributeUse required="false" inheritable="false" ref="C987"/>
      <scm:attributeUse required="false" inheritable="false" ref="C988"/>
      <scm:attributeUse required="false" inheritable="false" ref="C989"/>
      <scm:attributeUse required="false" inheritable="false" ref="C990"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C989"
                  name="_as"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C427"/>
   <scm:attribute id="C985"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C427"/>
   <scm:attribute id="C990"
                  name="_tunnel"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C427"/>
   <scm:attribute id="C987"
                  name="tunnel"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C427"/>
   <scm:attribute id="C988"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C427"/>
   <scm:attribute id="C986"
                  name="as"
                  type="C78"
                  global="false"
                  inheritable="false"
                  containingComplexType="C427"/>
   <scm:complexType id="C473"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C991"/>
      <scm:attributeUse required="false" inheritable="false" ref="C992" default="no"/>
      <scm:attributeUse required="false" inheritable="false" ref="C993"/>
      <scm:attributeUse required="false" inheritable="false" ref="C994"/>
      <scm:attributeUse required="false" inheritable="false" ref="C995"/>
      <scm:attributeUse required="false" inheritable="false" ref="C996"/>
      <scm:attributeUse required="false" inheritable="false" ref="C997"/>
      <scm:attributeUse required="false" inheritable="false" ref="C998"/>
      <scm:attributeUse required="false" inheritable="false" ref="C999"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1000"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@type | @_type) and exists(@validation | @_validation))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@href | @_href)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C995"
                  name="validation"
                  type="C94"
                  global="false"
                  inheritable="false"
                  containingComplexType="C473"/>
   <scm:attribute id="C996"
                  name="_href"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C473"/>
   <scm:attribute id="C994"
                  name="type"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C473"/>
   <scm:attribute id="C997"
                  name="_streamable"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C473"/>
   <scm:attribute id="C991"
                  name="href"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C473"/>
   <scm:attribute id="C998"
                  name="_use-accumulators"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C473"/>
   <scm:attribute id="C993"
                  name="use-accumulators"
                  type="C180"
                  global="false"
                  inheritable="false"
                  containingComplexType="C473"/>
   <scm:attribute id="C999"
                  name="_type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C473"/>
   <scm:attribute id="C683"
                  name="on-multiple-match"
                  type="C154"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C770"
                  name="_namespace-context"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C916"
                  name="_normalization-form"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C886"
                  name="doctype-system"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:wildcard id="C934" processContents="lax" constraint="any"/>
   <scm:attribute id="C740"
                  name="_doctype-public"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C549"
                  name="ordinal"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C790"
                  name="elements"
                  type="C0"
                  global="false"
                  inheritable="false"
                  containingComplexType="C377"/>
   <scm:wildcard id="C167"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/1999/XSL/Transform"/>
   <scm:attribute id="C912"
                  name="_include-content-type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C832"
                  name="_rollback-output"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C406"/>
   <scm:attribute id="C889"
                  name="html-version"
                  type="#decimal"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C687"
                  name="visibility"
                  type="C209"
                  global="false"
                  inheritable="false"
                  containingComplexType="C469"/>
   <scm:attribute id="C954"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C425"/>
   <scm:attribute id="C543"
                  name="level"
                  type="C234"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C663"
                  name="name"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C506"/>
   <scm:attribute id="C895"
                  name="normalization-form"
                  type="#NMTOKEN"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C768"
                  name="_base-uri"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:wildcard id="C314"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/1999/XSL/Transform"/>
   <scm:attribute id="C548"
                  name="letter-value"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:attribute id="C619"
                  name="elementFormDefault"
                  type="C245"
                  global="false"
                  inheritable="false"
                  containingComplexType="C475"/>
   <scm:attribute id="C583"
                  name="copy-accumulators"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C410"/>
   <scm:attribute id="C778"
                  name="_as"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C399"/>
   <scm:attribute id="C866"
                  name="error-code"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C389"/>
   <scm:attribute id="C977"
                  name="xpath"
                  type="C161"
                  global="false"
                  inheritable="false"
                  containingComplexType="C370"/>
   <scm:attribute id="C554"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C367"/>
   <scm:complexType id="C375"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C1001"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1002"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1003" default="yes"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1004" default=""/>
      <scm:attributeUse required="false" inheritable="false" ref="C1005"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1006"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1007"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1008"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1009"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1010"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1011"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1012"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@type | @_type) and exists(@validation | @_validation))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C1008"
                  name="_namespace"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C1010"
                  name="_use-attribute-sets"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C1005"
                  name="type"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C1009"
                  name="_inherit-namespaces"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C1011"
                  name="_type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C1002"
                  name="namespace"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C1003"
                  name="inherit-namespaces"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C1006"
                  name="validation"
                  type="C94"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C1007"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C1004"
                  name="use-attribute-sets"
                  type="C254"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C1001"
                  name="name"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:complexType id="C492"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C1013"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1014"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1015" default=""/>
      <scm:attributeUse required="false" inheritable="false" ref="C1016"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1017"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1018"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1019"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1020"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded">
         <scm:sequence>
            <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C123"/>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C123" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C123" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C1013"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C492"/>
   <scm:attribute id="C1019"
                  name="_use-attribute-sets"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C492"/>
   <scm:attribute id="C1020"
                  name="_visibility"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C492"/>
   <scm:attribute id="C1018"
                  name="_streamable"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C492"/>
   <scm:attribute id="C1014"
                  name="streamable"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C492"/>
   <scm:attribute id="C1016"
                  name="visibility"
                  type="C184"
                  global="false"
                  inheritable="false"
                  containingComplexType="C492"/>
   <scm:attribute id="C1017"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C492"/>
   <scm:attribute id="C966"
                  name="disable-output-escaping"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C485"/>
   <scm:attribute id="C1015"
                  name="use-attribute-sets"
                  type="C254"
                  global="false"
                  inheritable="false"
                  containingComplexType="C492"/>
   <scm:attribute id="C585"
                  name="type"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C410"/>
   <scm:attribute id="C932"
                  name="source"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C501"/>
   <scm:attribute id="C726"
                  name="omit-xml-declaration"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C326"
                  name="block"
                  type="C268"
                  global="false"
                  inheritable="false"
                  containingComplexType="C317"/>
   <scm:attribute id="C928"
                  name="_names"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C494"/>
   <scm:attribute id="C968"
                  name="_disable-output-escaping"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C485"/>
   <scm:attribute id="C597"
                  name="use-accumulators"
                  type="C180"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:complexType id="C373"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C1021"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1022" default="."/>
      <scm:attributeUse required="false" inheritable="false" ref="C1023" default=","/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C1024"
                        default="Infinity"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1025" default="-"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1026" default="e"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1027" default="NaN"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1028" default="%"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1029" default="~"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1030" default="0"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1031" default="#"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1032" default=";"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1033"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1034"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1035"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1036"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1037"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1038"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1039"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1040"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1041"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1042"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1043"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1044"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C1044"
                  name="_pattern-separator"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1035"
                  name="_grouping-separator"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1040"
                  name="_percent"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1027"
                  name="NaN"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1032"
                  name="pattern-separator"
                  type="C85"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1023"
                  name="grouping-separator"
                  type="C85"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1028"
                  name="percent"
                  type="C85"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1024"
                  name="infinity"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1033"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1036"
                  name="_infinity"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1043"
                  name="_digit"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1039"
                  name="_NaN"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1026"
                  name="exponent-separator"
                  type="C85"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1021"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1030"
                  name="zero-digit"
                  type="C84"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1031"
                  name="digit"
                  type="C85"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1042"
                  name="_zero-digit"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1029"
                  name="per-mille"
                  type="C85"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1034"
                  name="_decimal-separator"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1025"
                  name="minus-sign"
                  type="C85"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1037"
                  name="_minus-sign"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C1041"
                  name="_per-mille"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:complexType id="C395"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C1045"
                        default="interleave"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C1046"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
            <scm:edge term="C1046" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C1046" to="2"/>
         </scm:state>
         <scm:state nr="2" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C1045"
                  name="mode"
                  type="C306"
                  global="false"
                  inheritable="false"
                  containingComplexType="C395"/>
   <scm:element id="C1046"
                name="any"
                targetNamespace="http://www.w3.org/2001/XMLSchema"
                type="C294"
                global="false"
                containingComplexType="C395"
                nillable="false"
                abstract="false"/>
   <scm:attribute id="C227"
                  name="fixed"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C223"/>
   <scm:attribute id="C745"
                  name="_include-content-type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:complexType id="C421"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C426"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C426" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C426" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:complexType id="C409"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="empty">
      <scm:attributeUse required="false" inheritable="false" ref="C1047"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1048"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1049"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1050"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true"/>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $prefix in (@stylesheet-prefix, @result-prefix)/normalize-space(.)[. ne '#default']                            satisfies $prefix = in-scope-prefixes(.)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@stylesheet-prefix | @_stylesheet-prefix)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@result-prefix | @_result-prefix)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C1049"
                  name="_stylesheet-prefix"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C409"/>
   <scm:attribute id="C1048"
                  name="result-prefix"
                  type="C92"
                  global="false"
                  inheritable="false"
                  containingComplexType="C409"/>
   <scm:attribute id="C1047"
                  name="stylesheet-prefix"
                  type="C92"
                  global="false"
                  inheritable="false"
                  containingComplexType="C409"/>
   <scm:attribute id="C1050"
                  name="_result-prefix"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C409"/>
   <scm:complexType id="C837"
                    base="C278"
                    derivationMethod="restriction"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1051"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1052"/>
      <scm:attributeUse required="true" inheritable="false" ref="C279"/>
      <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C1052"
                  name="minOccurs"
                  type="#nonNegativeInteger"
                  global="false"
                  inheritable="false"
                  containingComplexType="C837">
      <scm:fixed>
         <scm:item type="#integer" value="1"/>
      </scm:fixed>
   </scm:attribute>
   <scm:attribute id="C1051"
                  name="maxOccurs"
                  type="#nonNegativeInteger"
                  global="false"
                  inheritable="false"
                  containingComplexType="C837">
      <scm:fixed>
         <scm:item type="#integer" value="1"/>
      </scm:fixed>
   </scm:attribute>
   <scm:attribute id="C706"
                  name="format"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C25"
                  name="default-mode"
                  type="C185"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:attribute id="C609"
                  name="_validation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C771"
                  name="_schema-aware"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C251"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C199"/>
   <scm:attribute id="C802"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C384"/>
   <scm:attribute id="C321"
                  name="substitutionGroup"
                  type="C267"
                  global="false"
                  inheritable="false"
                  containingComplexType="C317"/>
   <scm:wildcard id="C639"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/1999/XSL/Transform"/>
   <scm:attribute id="C577"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C391"/>
   <scm:attribute id="C712"
                  name="build-tree"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C622"
                  name="id"
                  type="#ID"
                  global="false"
                  inheritable="false"
                  containingComplexType="C475"/>
   <scm:attribute id="C955"
                  name="_match"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C425"/>
   <scm:wildcard id="C152" processContents="lax" constraint="any"/>
   <scm:attribute id="C701"
                  name="_flags"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C369"/>
   <scm:attribute id="C759"
                  name="xpath"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C766"
                  name="_xpath"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C644"
                  name="namespace"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C451"/>
   <scm:attribute id="C935"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C39"
                  name="_xpath-default-namespace"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C23"/>
   <scm:attribute id="C351"
                  name="final"
                  type="C346"
                  global="false"
                  inheritable="false"
                  containingComplexType="C347"/>
   <scm:attribute id="C707"
                  name="href"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C319"
                  name="final"
                  type="C346"
                  global="false"
                  inheritable="false"
                  containingComplexType="C317"/>
   <scm:attribute id="C340"
                  name="targetNamespace"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C317"/>
   <scm:attribute id="C733"
                  name="_format"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C992"
                  name="streamable"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C473"/>
   <scm:attribute id="C599"
                  name="type"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C870"
                  name="schemaLocation"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C401"/>
   <scm:attribute id="C612"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C423"/>
   <scm:attribute id="C273"
                  name="inheritable"
                  type="#boolean"
                  global="false"
                  inheritable="false"
                  containingComplexType="C272"/>
   <scm:wildcard id="C341"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C1012"
                  name="_validation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C375"/>
   <scm:attribute id="C717"
                  name="encoding"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C953"
                  name="collation"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C425"/>
   <scm:wildcard id="C72"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C734"
                  name="_href"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C76"
                  name="version"
                  type="#decimal"
                  global="false"
                  inheritable="false"
                  containingComplexType="C74"/>
   <scm:attribute id="C941"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C573"
                  name="_as"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C383"/>
   <scm:attribute id="C699"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C369"/>
   <scm:complexType id="C484"
                    base="C74"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C1053"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1054" default="yes"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C1055"
                        default="item()*"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1056"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1057"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1058"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1059"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1060"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1061"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1062"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1063"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1064"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1065"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1066"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1067"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1068"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C382"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C382" to="2"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C382" to="2"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $e in xsl:param satisfies empty($e/(@required | @_required))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $e in xsl:param satisfies empty($e/(@visibility | @_visibility))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="every $e in xsl:param satisfies (empty($e/(@select | @_select)) and empty($e/child::node()))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C1059"
                  name="new-each-time"
                  type="C260"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1066"
                  name="_override-extension-function"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1056"
                  name="visibility"
                  type="C184"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1065"
                  name="_streamability"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1057"
                  name="streamability"
                  type="C148"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1058"
                  name="override-extension-function"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1055"
                  name="as"
                  type="C78"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1063"
                  name="_as"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1064"
                  name="_visibility"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1054"
                  name="override"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1060"
                  name="cache"
                  type="C259"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1053"
                  name="name"
                  type="C149"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1061"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1068"
                  name="_cache"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1062"
                  name="_override"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:attribute id="C1067"
                  name="_identity-sensitive"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C484"/>
   <scm:wildcard id="C933"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:complexType id="C366"
                    base="C289"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="true" inheritable="false" ref="C1069"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1070"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
                  <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C518"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C13" to="1"/>
            <scm:edge term="C489" to="2"/>
            <scm:edge term="C414" to="2"/>
            <scm:edge term="C430" to="2"/>
            <scm:edge term="C507" to="2"/>
            <scm:edge term="C438" to="2"/>
            <scm:edge term="C498" to="2"/>
            <scm:edge term="C456" to="2"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C489" to="2"/>
            <scm:edge term="C414" to="2"/>
            <scm:edge term="C430" to="2"/>
            <scm:edge term="C507" to="2"/>
            <scm:edge term="C438" to="2"/>
            <scm:edge term="C498" to="2"/>
            <scm:edge term="C456" to="2"/>
         </scm:state>
         <scm:state nr="2" final="true">
            <scm:edge term="C489" to="2"/>
            <scm:edge term="C414" to="2"/>
            <scm:edge term="C430" to="2"/>
            <scm:edge term="C507" to="2"/>
            <scm:edge term="C438" to="2"/>
            <scm:edge term="C498" to="2"/>
            <scm:edge term="C456" to="2"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C1070"
                  name="id"
                  type="#ID"
                  global="false"
                  inheritable="false"
                  containingComplexType="C366"/>
   <scm:attribute id="C1069"
                  name="schemaLocation"
                  type="#anyURI"
                  global="false"
                  inheritable="false"
                  containingComplexType="C366"/>
   <scm:attribute id="C861"
                  name="mixed"
                  type="#boolean"
                  global="false"
                  inheritable="false"
                  containingComplexType="C433"/>
   <scm:wildcard id="C980"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:complexType id="C398"
                    base="C73"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false" inheritable="false" ref="C1071"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1072"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C77"/>
      <scm:elementParticle minOccurs="0" maxOccurs="unbounded" ref="C426"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C426" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C426" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C1072"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C398"/>
   <scm:attribute id="C1071"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C398"/>
   <scm:attribute id="C595"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C403"/>
   <scm:attribute id="C773"
                  name="name"
                  type="C186"
                  global="false"
                  inheritable="false"
                  containingComplexType="C399"/>
   <scm:attribute id="C831"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C406"/>
   <scm:attribute id="C628"
                  name="stable"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:complexType id="C496"
                    base="C8"
                    derivationMethod="extension"
                    abstract="false"
                    variety="element-only">
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C1073"
                        default="false"/>
      <scm:attributeUse required="false"
                        inheritable="false"
                        ref="C1074"
                        default="interleave"/>
      <scm:attributeUse required="false" inheritable="false" ref="C11"/>
      <scm:attributeWildcard ref="C12"/>
      <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
         <scm:sequence>
            <scm:elementParticle minOccurs="0" maxOccurs="1" ref="C13"/>
            <scm:modelGroupParticle minOccurs="1" maxOccurs="1">
               <scm:sequence>
                  <scm:elementParticle minOccurs="1" maxOccurs="1" ref="C1046"/>
               </scm:sequence>
            </scm:modelGroupParticle>
         </scm:sequence>
      </scm:modelGroupParticle>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0">
            <scm:edge term="C13" to="1"/>
            <scm:edge term="C1046" to="2"/>
         </scm:state>
         <scm:state nr="1">
            <scm:edge term="C1046" to="2"/>
         </scm:state>
         <scm:state nr="2" final="true"/>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C1074"
                  name="mode"
                  type="C261"
                  global="false"
                  inheritable="false"
                  containingComplexType="C496"/>
   <scm:attribute id="C1073"
                  name="appliesToEmpty"
                  type="#boolean"
                  global="false"
                  inheritable="false"
                  containingComplexType="C496"/>
   <scm:attribute id="C630"
                  name="_data-type"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C417"/>
   <scm:attribute id="C853"
                  name="as"
                  type="C78"
                  global="false"
                  inheritable="false"
                  containingComplexType="C381"/>
   <scm:attribute id="C283"
                  name="name"
                  type="#NCName"
                  global="false"
                  inheritable="false"
                  containingComplexType="C282"/>
   <scm:attribute id="C641"
                  name="notQName"
                  type="C215"
                  global="false"
                  inheritable="false"
                  containingComplexType="C419"/>
   <scm:attribute id="C723"
                  name="json-node-output-method"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:attribute id="C767"
                  name="_as"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C497"/>
   <scm:attribute id="C794"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C429"/>
   <scm:attribute id="C1022"
                  name="decimal-separator"
                  type="C85"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:wildcard id="C200"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C350"
                  name="abstract"
                  type="#boolean"
                  global="false"
                  inheritable="false"
                  containingComplexType="C347"/>
   <scm:attribute id="C645"
                  name="separator"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C451"/>
   <scm:attribute id="C975"
                  name="_names"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C388"/>
   <scm:attribute id="C963"
                  name="_as"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C487"/>
   <scm:attribute id="C1038"
                  name="_exponent-separator"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C373"/>
   <scm:attribute id="C537"
                  name="validation"
                  targetNamespace="http://www.w3.org/1999/XSL/Transform"
                  type="C94"
                  global="false"
                  inheritable="false"/>
   <scm:wildcard id="C194"
                 processContents="lax"
                 constraint="not"
                 namespaces="##local http://www.w3.org/2001/XMLSchema"/>
   <scm:attribute id="C75"
                  name="_version"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C74"/>
   <scm:attribute id="C885"
                  name="doctype-public"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C1000"
                  name="_validation"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C473"/>
   <scm:attribute id="C649"
                  name="_namespace"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C451"/>
   <scm:attribute id="C911"
                  name="_html-version"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C908"
                  name="_doctype-system"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C973"
                  name="visibility"
                  type="C184"
                  global="false"
                  inheritable="false"
                  containingComplexType="C388"/>
   <scm:complexType id="C392"
                    base="C344"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C1075"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1076" default="no"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1077"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1078"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1079"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1080"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeWildcard ref="C40"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
   </scm:complexType>
   <scm:attribute id="C1075"
                  name="select"
                  type="C156"
                  global="false"
                  inheritable="false"
                  containingComplexType="C392"/>
   <scm:attribute id="C1076"
                  name="terminate"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C392"/>
   <scm:attribute id="C1080"
                  name="_error-code"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C392"/>
   <scm:attribute id="C1078"
                  name="_select"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C392"/>
   <scm:attribute id="C1079"
                  name="_terminate"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C392"/>
   <scm:attribute id="C1077"
                  name="error-code"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C392"/>
   <scm:attribute id="C926"
                  name="visibility"
                  type="C308"
                  global="false"
                  inheritable="false"
                  containingComplexType="C494"/>
   <scm:attribute id="C826"
                  name="match"
                  type="C155"
                  global="false"
                  inheritable="false"
                  containingComplexType="C386"/>
   <scm:attribute id="C937"
                  name="order"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C442"/>
   <scm:attribute id="C902"
                  name="version"
                  type="#NMTOKEN"
                  global="false"
                  inheritable="false"
                  containingComplexType="C413"/>
   <scm:attribute id="C618"
                  name="attributeFormDefault"
                  type="C245"
                  global="false"
                  inheritable="false"
                  containingComplexType="C475"/>
   <scm:attribute id="C671"
                  name="as"
                  type="C78"
                  global="false"
                  inheritable="false"
                  containingComplexType="C464"/>
   <scm:attribute id="C580"
                  name="_use-character-maps"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C391"/>
   <scm:attribute id="C754"
                  name="_undeclare-prefixes"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C454"/>
   <scm:complexType id="C368"
                    base="C95"
                    derivationMethod="extension"
                    abstract="false"
                    variety="mixed">
      <scm:attributeUse required="false" inheritable="false" ref="C1081"/>
      <scm:attributeUse required="false" inheritable="false" ref="C1082"/>
      <scm:attributeUse required="false" inheritable="false" ref="C75"/>
      <scm:attributeUse required="false" inheritable="false" ref="C33"/>
      <scm:attributeUse required="false" inheritable="false" ref="C97"/>
      <scm:attributeUse required="false" inheritable="false" ref="C28"/>
      <scm:attributeUse required="false" inheritable="false" ref="C38"/>
      <scm:attributeUse required="false" inheritable="false" ref="C24"/>
      <scm:attributeUse required="false" inheritable="false" ref="C76"/>
      <scm:attributeUse required="false" inheritable="false" ref="C35"/>
      <scm:attributeUse required="false" inheritable="false" ref="C29"/>
      <scm:attributeUse required="false" inheritable="false" ref="C25"/>
      <scm:attributeUse required="false" inheritable="false" ref="C32"/>
      <scm:attributeUse required="false" inheritable="false" ref="C98"/>
      <scm:attributeUse required="false" inheritable="false" ref="C37"/>
      <scm:attributeUse required="false" inheritable="false" ref="C30"/>
      <scm:attributeUse required="false" inheritable="false" ref="C26" default="strip"/>
      <scm:attributeUse required="false" inheritable="false" ref="C34"/>
      <scm:attributeUse required="false" inheritable="false" ref="C39"/>
      <scm:attributeUse required="false" inheritable="false" ref="C31"/>
      <scm:attributeUse required="false" inheritable="false" ref="C27"/>
      <scm:attributeUse required="false" inheritable="false" ref="C36"/>
      <scm:attributeWildcard ref="C99"/>
      <scm:modelGroupParticle minOccurs="0" maxOccurs="unbounded" ref="C100"/>
      <scm:finiteStateMachine initialState="0">
         <scm:state nr="0" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
         <scm:state nr="1" final="true">
            <scm:edge term="C101" to="1"/>
            <scm:edge term="C102" to="1"/>
            <scm:edge term="C103" to="1"/>
            <scm:edge term="C104" to="1"/>
            <scm:edge term="C105" to="1"/>
            <scm:edge term="C106" to="1"/>
            <scm:edge term="C107" to="1"/>
            <scm:edge term="C108" to="1"/>
            <scm:edge term="C109" to="1"/>
            <scm:edge term="C110" to="1"/>
            <scm:edge term="C111" to="1"/>
            <scm:edge term="C112" to="1"/>
            <scm:edge term="C113" to="1"/>
            <scm:edge term="C114" to="1"/>
            <scm:edge term="C115" to="1"/>
            <scm:edge term="C116" to="1"/>
            <scm:edge term="C117" to="1"/>
            <scm:edge term="C118" to="1"/>
            <scm:edge term="C119" to="1"/>
            <scm:edge term="C120" to="1"/>
            <scm:edge term="C121" to="1"/>
            <scm:edge term="C122" to="1"/>
            <scm:edge term="C123" to="1"/>
            <scm:edge term="C124" to="1"/>
            <scm:edge term="C125" to="1"/>
            <scm:edge term="C126" to="1"/>
            <scm:edge term="C127" to="1"/>
            <scm:edge term="C128" to="1"/>
            <scm:edge term="C129" to="1"/>
            <scm:edge term="C130" to="1"/>
            <scm:edge term="C131" to="1"/>
            <scm:edge term="C132" to="1"/>
            <scm:edge term="C133" to="1"/>
            <scm:edge term="C134" to="1"/>
            <scm:edge term="C135" to="1"/>
            <scm:edge term="C136" to="1"/>
            <scm:edge term="C137" to="1"/>
            <scm:edge term="C138" to="1"/>
            <scm:edge term="C139" to="1"/>
            <scm:edge term="C140" to="1"/>
            <scm:edge term="C141" to="1"/>
            <scm:edge term="C142" to="1"/>
            <scm:edge term="C143" to="1"/>
            <scm:edge term="C144" to="1"/>
            <scm:edge term="C145" to="1"/>
         </scm:state>
      </scm:finiteStateMachine>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="not(exists(@select | @_select) and (exists(* except xsl:fallback) or exists(text()[normalize-space()])))"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
      <scm:assertion xmlns:xs="http://www.w3.org/2001/XMLSchema"
                     xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                     xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
                     test="exists(@name | @_name)"
                     defaultNamespace=""
                     xml:base="file:/Users/mike/w3c/xslt30-test-B/tests/misc/catalog/schema-for-xslt30.xsd"/>
   </scm:complexType>
   <scm:attribute id="C1081"
                  name="name"
                  type="C342"
                  global="false"
                  inheritable="false"
                  containingComplexType="C368"/>
   <scm:attribute id="C1082"
                  name="_name"
                  type="#string"
                  global="false"
                  inheritable="false"
                  containingComplexType="C368"/>
</scm:schema>
<?Σ 11e967e8?>
